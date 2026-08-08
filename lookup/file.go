package lookup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// relativePath returns the proper relative path for the given file path. If
// the relativeTo string equals "-", then it means that it's from the stdin,
// and the returned path will be the current working directory. Otherwise, if
// file is really an absolute path, then it will be returned without any
// changes. Otherwise, the returned path will be a combination of relativeTo
// and file.
func relativePath(file, relativeTo string) string {
	// stdin: return the current working directory if possible.
	if relativeTo == "-" {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, file)
		}
	}

	// If the given file is already an absolute path, just return it.
	// Otherwise, the returned path will be relative to the given relativeTo
	// path.
	if filepath.IsAbs(file) {
		return file
	}

	abs, err := filepath.Abs(filepath.Join(filepath.Dir(relativeTo), file))
	if err != nil {
		logrus.Errorf("Failed to get absolute directory: %s", err)
		return file
	}
	return abs
}

// FileResourceLookup implements the project.ResourceLookup interface. Its zero
// value denies local file reads. Call NewFileResourceLookup with the locally
// selected Compose files to permit reads beneath those files' directories.
type FileResourceLookup struct {
	roots             []string
	initializationErr error
}

// NewFileResourceLookup creates a local resource reader rooted at the
// directories containing the selected Compose files. Passing no files creates
// a deny-by-default reader for in-memory Compose input.
func NewFileResourceLookup(composeFiles ...string) *FileResourceLookup {
	lookup := &FileResourceLookup{}
	for _, composeFile := range composeFiles {
		if composeFile == "" {
			continue
		}

		var root string
		if composeFile == "-" {
			cwd, err := os.Getwd()
			if err != nil {
				lookup.initializationErr = fmt.Errorf("determine resource root for stdin: %w", err)
				return lookup
			}
			root = cwd
		} else {
			absoluteFile, err := filepath.Abs(composeFile)
			if err != nil {
				lookup.initializationErr = fmt.Errorf("determine resource root for %q: %w", composeFile, err)
				return lookup
			}
			root = filepath.Dir(absoluteFile)
		}

		resolvedRoot := filepath.Clean(root)
		duplicate := false
		for _, existing := range lookup.roots {
			if existing == resolvedRoot {
				duplicate = true
				break
			}
		}
		if !duplicate {
			lookup.roots = append(lookup.roots, resolvedRoot)
		}
	}
	return lookup
}

func resourcePath(file, relativeTo string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("local resource path is empty")
	}
	if filepath.IsAbs(file) {
		return filepath.Clean(file), nil
	}

	base := relativeTo
	if relativeTo == "-" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("determine resource path for stdin: %w", err)
		}
		base = filepath.Join(cwd, "stdin-compose.yml")
	}

	absolute, err := filepath.Abs(filepath.Join(filepath.Dir(base), file))
	if err != nil {
		return "", fmt.Errorf("resolve local resource %q: %w", file, err)
	}
	return filepath.Clean(absolute), nil
}

func relativeToRoot(root, target string) (string, bool) {
	relative, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return relative, true
}

// Lookup returns the content and the actual filename of the file that is "built" using the
// specified file and relativeTo string. file and relativeTo are supposed to be file path.
// If file starts with a slash ('/'), it tries to load it, otherwise it will build a
// filename using the folder part of relativeTo joined with file.
func (f *FileResourceLookup) Lookup(file, relativeTo string) ([]byte, string, error) {
	if f == nil {
		return nil, "", fmt.Errorf("local resource lookup is not configured")
	}
	if f.initializationErr != nil {
		return nil, "", f.initializationErr
	}
	if len(f.roots) == 0 {
		return nil, "", fmt.Errorf("local file references are disabled for in-memory Compose input")
	}

	target, err := resourcePath(file, relativeTo)
	if err != nil {
		return nil, "", err
	}

	for _, allowedRoot := range f.roots {
		relative, allowed := relativeToRoot(allowedRoot, target)
		if !allowed {
			continue
		}

		root, err := os.OpenRoot(allowedRoot)
		if err != nil {
			return nil, "", fmt.Errorf("open configured resource root: %w", err)
		}
		resource, openErr := root.Open(relative)
		if openErr != nil {
			root.Close()
			return nil, target, fmt.Errorf("open local resource %q: %w", file, openErr)
		}

		logrus.Debugf("Reading local resource %s", target)
		contents, readErr := io.ReadAll(resource)
		closeErr := resource.Close()
		rootCloseErr := root.Close()
		if readErr != nil {
			return nil, target, fmt.Errorf("read local resource %q: %w", file, readErr)
		}
		if closeErr != nil {
			return nil, target, fmt.Errorf("close local resource %q: %w", file, closeErr)
		}
		if rootCloseErr != nil {
			return nil, target, fmt.Errorf("close configured resource root: %w", rootCloseErr)
		}
		return contents, target, nil
	}

	return nil, target, fmt.Errorf("local resource %q is outside the configured Compose directories", file)
}

// ResolvePath returns the path to be used for the given path volume. This
// function already takes care of relative paths.
func (f *FileResourceLookup) ResolvePath(path, relativeTo string) string {
	vs := strings.SplitN(path, ":", 2)
	if len(vs) != 2 || filepath.IsAbs(vs[0]) {
		return path
	}
	vs[0] = relativePath(vs[0], relativeTo)
	return strings.Join(vs, ":")
}
