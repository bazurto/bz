
deps = [
        "github.com/bazurto/groovy",
        "github.com/bazurto/python@3",
]

triggers  {
  preRun = "$DIR/prerun.py"
  install = "$DIR/install.py"
}
