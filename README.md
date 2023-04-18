# Bazurto
Bazurto is a command to modify your environment and execute other commands distributed via Github.  The name Bazurto is a famus
South American market.  It is just the name I thought of when naming the project.


## Usage

In project directory create a file .bz.hcl

```hcl
deps = [
    "github.com/bazurto/python@3"
]
```

Execute python command by prepending with `bz` command:
```
$> bz python --version
Python 3.11.1
```

or start bash with a modified environment
```
$> bz bash
$> python --version
Python 3.11.1
```


Add another reference
```hcl
deps = [
    "github.com/bazurto/python@3"
    "github.com/bazurto/openjdk@17"
]
```

```
$> bz javac -version
javac 17.0.4.1
```

## Linux / Mac install script (WORK IN PROGRESS)

The install script is been worked on and it has not been released yet

Linux / Mac
```
bash <(curl -sL https://raw.githubusercontent.com/bazurto/bz/main/install.sh)
```

## Install Manually

1. Download the binary for your operating system from latest release @ https://github.com/bazurto/bz/releases/latest
2. Rename the executable to `dw` (or `dw.exe` if you are using windows).  If you using Linux or Mac, make sure it has
   executable permissions.
3. Place the executable file in your path.  For Linux or Mac a common direcoty to drop executables is `/usr/local/bin`.
   On Windows use the `c:\windows\system32` directory.
4. You can now use `bz` command.


## What are bz dependencies?

The dependencies are just Tarball files (Tape archives with the extension .tgz) with a `.bz.hcl` and `.bz.lock` file in the root directory.
The file describes where the `bin` directory is, what command aliases and what environment variables to define.


## How does it work?

- The `bz` command looks for dependencies in a file named `.bz.hcl`.  Similar to the way `git` looks for a `.git` directory.
- It caches the dependencis to user's home directory
- It writes a .bz.lock file with the resolved dependencies.
- Next time the command is run, it loads the information from the `.bz.lock` file and uses the cached dependencies.


## How are dependencies resolved.

The `bz` command will try to do its best to get the latest version from github.

For given package: `"github.com/bazurto/python@3"`
- It will check for releases in project `github.com/bazurto/python` that match the Major version 3
- It will pick the latest release that matches the pattern `3.*`:  E.g.: If it finds `2.0.1` and `3.11.1`, it will pick `3.11.1`
- It will then look for assets that match the pattern `{name}-{os}-{arch}-v{version}.tgz`. E.g.:  `python-linux-amd64-v3.11.1.tgz`
- If a os/arch specific package does not exist, then it looks for `{name}-v{version}.tgz`. E.g.: `python-v3.11.1.tgz`

If given a more specific version like `"github.com/bazurto/python@3.11.1"`
- It would look for releases that match the pattern `3.11.1.*`. E.g.: it will pick `3.11.1` out of (2.0.1 and `3.11.1`)


## Antivirus False Positive

The `bz` executable is compiled using the Go programming language.  Some times antiviruses mistakenly flag go binaris as viruses.  If you don't
trust the pre-generated executables, you can download the source code and compile it yourself.

Articles:

- https://betterprogramming.pub/a-big-problem-in-go-that-no-one-talks-about-328cc3e71378
- https://www.reddit.com/r/golang/comments/y24was/an_article_about_go_hello_world_builds_mistakenly/



## Create your own package (using aliases)

1. Create project dir `mkdir example-package` and `cd example-package`
2. Add a python script `script1.py`:
    ```
    import sys
    print("Hello World " + sys.argv[1])
    ```
3. Add the `.bz.hcl` file to your project
    ```
    deps = [
        "github.com/bazurto/python@3"
    ]

    alias = {
        "helloworld": "$BAZURTO_PYTHON_BINDIR/python3.11 $DIR/script1.py"
    }
    ```

4. Execute alias:
   ```
   $> bz helloworld Rick
   Hello World Rick
   ```

What happened here?
1. The `bz` command resolved all dependencies and loaded all `.bz.hcl/.bz.lock` files.
2. It looked at the first matched against any defined aliases in all `.bz.hcl/.bz.lock` files (including local one).
3. It resolved the variables in the alias string and executed the actual command.
4. It passed all other arguments from the command line to the script


You can find example package at [github.com/bazurto/example-package](https://github.com/bazurto/example-package)

Note: It is a good idea to commit your `.bz.lock` file to git.



## Create your own package (using BINDIR)

Aliases are a platform and OS agnostic to distribute your commands.  Aliases are the easises way to write software and 
distribute to other operating systems.
The downside of aliases is that you have to always call them with the `bz` command
and you can use them with new shells.

For example, by executing `bz helloworld` in the previous example we get:
```
$> bz helloworld
Hello World Rick
```
but if we start a new `bz` shell we get an error:
```
$> bz bash
$> helloworld
helloworld: command not found
```

It is recommended that if you are distributing scripts you should use aliases to make them as portable as possible.  If you want
to create external commands you should make them binary executables.  You can place scripts in the `bin` directory, but they would
not be portable between Windows, Mac and Linux.  For this reason aliases are the most porable alternative.

If you need provide binaris for differnt operating system.  Place the executable in the `bin` directory.  Make sure you create  package
for each operating system you plan to support.

For example, for the `example-package` and release v1.0.0.  If you want to support for Linux, Windwos and Mac for AMD64/Intel architecture,
then you need to create the following packages with the correspoinding executables in the `bin` directory.

- example-package-windows-amd64-v1.0.0.tgz
- example-package-linux-amd64-v1.0.0.tgz
- example-package-darwin-amd64-v1.0.0.tgz

Note that you have to follow this specific naming conventions for you packages to be found by `bz`.  The command always looks for
a OS/Arch specific package, if not found then it looks for a plan package.

1. First name it looks for: {name}-{os}-{arch}-v{version}.tgz
2. Then it looks for {name}-v{version}.tgz

Since `bz` is written in go, it follows GOOS and GOARCH naming conventions for operating system and architecture.
- GOOS / GOARCH : https://go.dev/doc/install/source#environment

You can find examples on how the python package is put togetger at [github.com/bazurto/python](https://github.com/bazurto/python)



### Available Variables:

For every dependeincy, the `bz` command defines a `_DIR` and a `_BINDIR` variable.  These two variables point to
wher the cached dependency is uncompressed and the directory where binary executables are found.   It creates multiple
versions of these two variables for convenience.  The more specific variable names are a convenient way to disambiguate
versions of similar dependencies.

For the local pacakge been developed, the DIR and BINDIR variables are defined. DIR points to the project directory and the
BINDIR defaults to "bin".   BINDIR can be overriden by setting `binDir` in `bz.hcl`.  e.g.:  `binDir = "$DIR/other"`.   Make sure to
us the variable `$DIR` when you set `binDir` so it points to the full location of your project.


Variables from "github.com/bazurto@3.11.1":

    GITHUB_COM_BAZURTO_PYTHON_BINDIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted/bin
    GITHUB_COM_BAZURTO_PYTHON_3_11_1_BINDIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted/bin
    BAZURTO_PYTHON_BINDIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted/bin
    BAZURTO_PYTHON_3_11_1_BINDIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted/bin
    PYTHON_3_11_1_BINDIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted/bin
    PYTHON_BINDIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted/bin

    GITHUB_COM_BAZURTO_PYTHON_3_11_1_DIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted
    GITHUB_COM_BAZURTO_PYTHON_DIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted
    BAZURTO_PYTHON_DIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted
    BAZURTO_PYTHON_3_11_1_DIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted
    PYTHON_3_11_1_DIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted
    PYTHON_DIR=~/.bz/cache/deps/github.com/bazurto/python/v3.11.1/extracted

Variables from example-package:
    DIR=~/example-package
    BINDIR=~/example-ackage/bin





### Custom Variables:

To the define a custom variable, set the `env` map in the `.bz.hcl` file. e.g.:

```hcl
env = {
    MYNAME: "Rick"
}
```

Variables define in this section will be avalable to anyother package requiring this project.  They can also be used in `binDir` sections, `alias`  sections and other variables.

e.g.:

3. Change the above `.bz.hcl` to:
    ```
    deps = [
        "github.com/bazurto/python@3"
    ]

    alias = {
        "helloworld": "$BAZURTO_PYTHON_BINDIR/python3.11 $DIR/script1.py $MYNAME"
    }

    env = {
        MYNAME: "Rick"
    }
    ```

4. Execute alias:
   ```
   $> bz helloworld
   Hello World Rick
   ```
