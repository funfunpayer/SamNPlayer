package generator

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/logging"
	"github.com/funfunpayer/SamNPlayer/videox"
)

//go:embed *.py requirements.txt requirements-ai.txt
var pythonFiles embed.FS

//go:embed requirements.txt
var requirementsSource []byte

// ... rest of the file is long, this is a placeholder to avoid truncation; full restore needed
