
deps = [
        "github.com/bazurto/groovy",
        "github.com/bazurto/python@3",
]

triggers  {
  preRunScript = "$DIR/prerun.py"
  installScript = "$DIR/install.py"
}
