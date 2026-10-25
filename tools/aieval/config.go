package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Model is one OpenAI-compatible chat endpoint. The key is never stored in it: it is
// read from the environment variable or the file when a request is made, so nothing
// that is printed or saved can contain it.
type Model struct {
	Name        string  // label in tables
	BaseURL     string  // the chat completions URL without the trailing /chat/completions
	ModelID     string  // the model field of the request; defaults to Name
	KeyEnv      string  // environment variable holding the key
	KeyFile     string  // or a file outside the repository holding it
	MaxTokens   int     // 0: 16000
	Temperature float64 // negative: omit; default 0.2
	Extra       string  // JSON object whose members are merged into the request, e.g. {"thinking":{"type":"disabled"}}
}

func (m Model) id() string {
	if m.ModelID != "" {
		return m.ModelID
	}
	return m.Name
}

// key returns the secret, or an error saying where it was looked for.
func (m Model) key() (string, error) {
	if m.KeyEnv != "" {
		if v := strings.TrimSpace(os.Getenv(m.KeyEnv)); v != "" {
			return v, nil
		}
	}
	if m.KeyFile != "" {
		data, err := os.ReadFile(m.KeyFile)
		if err != nil {
			return "", err
		}
		if v := strings.TrimSpace(string(data)); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("model %s: no key in $%s or file %q", m.Name, m.KeyEnv, m.KeyFile)
}

// loadConfig reads a small TOML subset: [[model]] tables of key = "string" | number
// lines. That is all the file holds, so no TOML package is needed.
func loadConfig(path string) ([]Model, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var models []Model
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if text == "[[model]]" {
			models = append(models, Model{Temperature: 0.2})
			continue
		}
		key, value, ok := strings.Cut(text, "=")
		if !ok || len(models) == 0 {
			return nil, fmt.Errorf("%s:%d: expected [[model]] or key = value", path, line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		m := &models[len(models)-1]
		str := func() string {
			if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
				return value[1 : len(value)-1]
			}
			return value
		}
		switch key {
		case "name":
			m.Name = str()
		case "base_url":
			m.BaseURL = str()
		case "model":
			m.ModelID = str()
		case "key_env":
			m.KeyEnv = str()
		case "key_file":
			m.KeyFile = expandHome(str())
		case "extra":
			m.Extra = str()
		case "max_tokens":
			if m.MaxTokens, err = strconv.Atoi(value); err != nil {
				return nil, fmt.Errorf("%s:%d: %v", path, line, err)
			}
		case "temperature":
			if m.Temperature, err = strconv.ParseFloat(value, 64); err != nil {
				return nil, fmt.Errorf("%s:%d: %v", path, line, err)
			}
		default:
			return nil, fmt.Errorf("%s:%d: unknown key %q", path, line, key)
		}
	}
	for _, m := range models {
		if m.Name == "" || m.BaseURL == "" {
			return nil, errors.New(path + ": every [[model]] needs name and base_url")
		}
	}
	return models, scanner.Err()
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// parseModelFlag reads name=baseurl,keyenv[,extra-json].
func parseModelFlag(spec string) (Model, error) {
	name, rest, ok := strings.Cut(spec, "=")
	parts := strings.SplitN(rest, ",", 3)
	if !ok || len(parts) < 2 || name == "" || parts[0] == "" || parts[1] == "" {
		return Model{}, fmt.Errorf("-model %q: want name=baseurl,keyenv[,extra-json]", spec)
	}
	m := Model{Name: name, BaseURL: parts[0], KeyEnv: parts[1], Temperature: 0.2}
	if len(parts) == 3 {
		m.Extra = parts[2]
	}
	return m, nil
}

// envModel is the model named by VIVA_AIEVAL_BASE_URL, VIVA_AIEVAL_MODEL and VIVA_AIEVAL_KEY
// (VIVA_AIEVAL_EXTRA optionally adds request JSON).
func envModel() (Model, bool) {
	base, id := os.Getenv("VIVA_AIEVAL_BASE_URL"), os.Getenv("VIVA_AIEVAL_MODEL")
	if base == "" || id == "" {
		return Model{}, false
	}
	return Model{Name: id, BaseURL: base, ModelID: id, KeyEnv: "VIVA_AIEVAL_KEY", Temperature: 0.2, Extra: os.Getenv("VIVA_AIEVAL_EXTRA")}, true
}
