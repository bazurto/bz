deps = [
        "github.com/bazurto/groovy",
        "github.com/bazurto/python@3",
]

triggers  {
  preRunScript = "$DIR/prerun.php"
  installScript = "$DIR/install.php"
}
