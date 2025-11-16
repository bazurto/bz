package resolver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	pb "github.com/bazurto/bz/grpc/bazurto" // Adjust the import path as necessary
	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type BazurtoResolver struct {
	appCtx  *model.AppContext
	host    string
	port    int
	token   string
	timeout time.Duration
}

type BazurtoResolverOptions func(*BazurtoResolver)

func WithHost(host string) BazurtoResolverOptions {
	return func(o *BazurtoResolver) {
		o.host = host
	}
}
func WithPort(port int) BazurtoResolverOptions {
	return func(o *BazurtoResolver) {
		o.port = port
	}
}
func WithToken(token string) BazurtoResolverOptions {
	return func(o *BazurtoResolver) {
		o.token = token
	}
}
func WithTimeout(timeout time.Duration) BazurtoResolverOptions {
	return func(o *BazurtoResolver) {
		o.timeout = timeout
	}
}

func NewBazurtoResolver(
	appCtx *model.AppContext,
	options ...BazurtoResolverOptions,
) *BazurtoResolver {
	r := &BazurtoResolver{
		appCtx:  appCtx,
		host:    "localhost",
		port:    50051,
		timeout: time.Second * 5,
		token:   "12345",
	}
	for _, opt := range options {
		opt(r)
	}
	return r
}

func (o *BazurtoResolver) newRpc(f func(context.Context, pb.BazurtoClient) error) error {
	clientConn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", o.host, o.port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	ctx, cancel := context.WithTimeout(
		metadata.NewOutgoingContext(context.Background(),
			metadata.New(map[string]string{"bztoken": o.token}),
		), o.timeout,
	)
	defer cancel()

	c := pb.NewBazurtoClient(clientConn)
	return f(ctx, c)
}

func (o *BazurtoResolver) String() string {
	return "BazurtoResolver{}"
}

func (o *BazurtoResolver) ResolveCoord(c model.FuzzyCoord) (*model.LockedCoord, error) {
	Debug.Printf("Start BazurtoResolver.ResolveCoord(%s)", c.String())

	if c.URL.Scheme != "https" &&
		c.URL.Scheme != "http" &&
		c.URL.Hostname() != "" &&
		c.URL.Hostname() != "bz" &&
		c.URL.Hostname() != "bazurto" {
		Debug.Printf("BazurtoResolver.ResolveCoord: unsupported scheme: %s", c.URL.Scheme)
		return nil, nil
	}

	var resolvedUrl *url.URL
	err := o.newRpc(func(ctx context.Context, client pb.BazurtoClient) error {
		resp, err := client.ResolveCoord(ctx, &pb.ResolveCoordRequest{
			Coord: c.String(),
		})
		if err != nil {
			log.Fatalf("Error calling ResolveCoord: %v", err)
		}
		u, err := url.Parse(resp.Coord)
		if err != nil {
			return err
		}
		resolvedUrl = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	if resolvedUrl == nil {
		return nil, nil
	}

	//
	version := model.NewVersion(resolvedUrl.Fragment)
	lc, err := model.NewLockedCoord(
		resolvedUrl.Scheme,
		resolvedUrl.Host,
		resolvedUrl.Path,
		version,
		nil,
	)
	return &lc, err
}

func (o *BazurtoResolver) DownloadResolvedCoord(lc model.LockedCoord) (string, error, bool) {
	if lc.URL.Scheme != "https" &&
		lc.URL.Scheme != "http" &&
		lc.URL.Hostname() != "" &&
		lc.URL.Hostname() != "bz" &&
		lc.URL.Hostname() != "bazurto" {
		Debug.Printf("BazurtoResolver.ResolveCoord: unsupported scheme: %s", lc.URL.Scheme)
		return "", nil, false
	}

	// where to download and extract
	tmp := strings.Split(strings.Trim(lc.URL.Path, "/"), "/")

	tmp = append([]string{o.appCtx.UserCacheDirName, "deps"}, tmp...)
	tmp = append(tmp, fmt.Sprintf("v%s", lc.URL.Fragment))
	dir := filepath.Join(tmp...)                    // ./cache/deps/owner/repo/vX.Y.Z
	extractToDir := filepath.Join(dir, "extracted") // ./cache/deps/owner/repo/vX.Y.Z/extracted

	// nothing to do... already installed
	if utils.FileExists(extractToDir) {
		return extractToDir, nil, true
	}

	if err := utils.MkdirIfNotExists(dir); err != nil {
		return "", err, false
	} else {
		Debug.Printf("dir already exists: %s", dir)
	}

	//
	var file string
	err := o.newRpc(func(ctx context.Context, client pb.BazurtoClient) error {
		r, err := client.ResolveCoord(ctx, &pb.ResolveCoordRequest{Coord: lc.String()})
		if err != nil {
			return err
		}
		if r == nil {
			return fmt.Errorf("unable to resolve coord: %s", lc.String())
		}

		var match *pb.Artifact
		for _, a := range r.Artifacts {
			if a.Os == runtime.GOOS &&
				a.Arch == runtime.GOARCH {
				match = a
				break
			} else if a.Os == "" && a.Arch == "" {
				match = a // fallback to generic asset
			}
		}
		if match == nil {
			return fmt.Errorf("no matching artifact found for %s/%s in %s", runtime.GOOS, runtime.GOARCH, lc.String())
		}

		stream, err := client.DownloadCoord(ctx, &pb.DownloadCoordRequest{
			Coord:        lc.String(),
			ArtifactName: match.Name,
		})
		if err != nil {
			return err
		}
		file = filepath.Join(dir, match.Name) // Set the outter file name path
		out, err := os.Create(file)
		if err != nil {
			return err
		}
		defer out.Close()

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if _, err := out.Write(chunk.Data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err, false
	}

	err = ExtractDependency(file, extractToDir)
	if err != nil {
		return "", fmt.Errorf("unable to extract dependency: %w", err), false
	}

	return extractToDir, nil, true
}
