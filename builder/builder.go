package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

//BuildConfig describe the build to be performed
type BuildConfig struct {
	GoPath, GoRoot, Path string // environment
	SourceFolder         string // Folder hosting the source code
	Name                 string // Name of the built program
}

//UseShellEnv reuse current shell environment for build
func (b *BuildConfig) UseShellEnv() {
	b.GoPath = os.Getenv("GOPATH")
	b.GoRoot = os.Getenv("GOROOT")
	b.Path = os.Getenv("PATH")
}

func (b *BuildConfig) commandEnv() []string {
	return []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"GOPATH=" + b.GoPath,
		"GOROOT=" + b.GoRoot,
		"PATH=" + b.Path,
		"CGO_ENABLED=0",
	}
}

//Build launch the gobuild
func (b *BuildConfig) Build() (string, error) {
	tmpDir, err := ioutil.TempDir("", b.Name)
	if err != nil {
		return "", fmt.Errorf("can't create temporary directory: %s", err)
	}

	output := filepath.Join(tmpDir, b.Name)
	//ldflags := `-extldflags "-static"`
	// command := []string{
	// 	"go", "build", "-o", output, "-a", "--ldflags",
	// 	ldflags, "-tags", "netgo",
	// 	"-installsuffix", "netgo", ".",
	// }
	command := []string{
		"go", "build", "-o", output, "-a", "-installsuffix", "cgo", "-ldflags", "'-s'", b.SourceFolder,
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = b.commandEnv()

	data, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(data))
		return "", err
	}
	return output, nil
}
