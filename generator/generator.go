//go:embed *.py requirements.txt requirements-ai.txt
var pythonFiles embed.FS

//go:embed requirements.txt
var requirementsSource []byte
