package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsmccain/plex-preroll/internal"
	"github.com/joho/godotenv"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type prompts struct {
	reader *bufio.Reader
	input  io.Reader
	output io.Writer
}

func (p *prompts) line(prompt string) (string, error) {
	fmt.Fprint(p.output, prompt)
	value, err := p.reader.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && len(value) > 0) {
		return "", fmt.Errorf("reading setup input: %w", err)
	}
	return strings.TrimSpace(value), nil
}

func (p *prompts) token() (string, error) {
	for {
		var value string
		var err error
		if file, ok := p.input.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
			fmt.Fprint(p.output, "Enter your PLEX_TOKEN (input hidden): ")
			var password []byte
			password, err = term.ReadPassword(int(file.Fd()))
			fmt.Fprintln(p.output)
			value = strings.TrimSpace(string(password))
		} else {
			value, err = p.line("Enter your PLEX_TOKEN: ")
		}
		if err != nil {
			return "", fmt.Errorf("reading Plex token: %w", err)
		}
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(p.output, "Plex token cannot be empty.")
	}
}

func (p *prompts) yesNo(prompt string) (bool, error) {
	for {
		value, err := p.line(prompt)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(value) {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		}
		fmt.Fprintln(p.output, "Enter y or n.")
	}
}

func (p *prompts) date(prompt string) (string, error) {
	for {
		value, err := p.line(prompt)
		if err != nil {
			return "", err
		}
		date, err := internal.NormalizeMMDD(value)
		if err == nil {
			return date, nil
		}
		fmt.Fprintln(p.output, "Enter a valid month and day, such as 10-31.")
	}
}

func (p *prompts) savePath(prompt, defaultPath string) (string, error) {
	for {
		value, err := p.line(fmt.Sprintf("%s (default: %s): ", prompt, defaultPath))
		if err != nil {
			return "", err
		}
		if value == "" {
			value = defaultPath
		}
		expanded, err := internal.ExpandHomePath(value)
		if err != nil {
			fmt.Fprintln(p.output, err)
			continue
		}
		// Scheduled jobs may use a different working directory.
		absolute, err := filepath.Abs(expanded)
		if err != nil {
			return "", err
		}
		return resolveParent(absolute)
	}
}

// Resolve existing parent symlinks so aliases cannot point both outputs at the
// same path. Leave the final component alone so confirmOverwrite rejects links.
func resolveParent(path string) (string, error) {
	parent := filepath.Dir(path)
	parts := []string{filepath.Base(path)}
	for {
		resolved, err := filepath.EvalSymlinks(parent)
		if err == nil {
			return filepath.Join(append([]string{resolved}, parts...)...), nil
		}
		if !os.IsNotExist(err) || parent == filepath.Dir(parent) {
			return "", err
		}
		parts = append([]string{filepath.Base(parent)}, parts...)
		parent = filepath.Dir(parent)
	}
}

func (p *prompts) files(prompt string) ([]string, error) {
	files := []string{}
	for {
		value, err := p.line(prompt)
		if err != nil {
			return nil, err
		}
		if value == "" {
			return files, nil
		}
		path, err := internal.ExpandHomePath(value)
		if err != nil || !filepath.IsAbs(path) {
			fmt.Fprintln(p.output, "Use an absolute path or a path beginning with ~/.")
			continue
		}
		if strings.ContainsAny(path, ",;") {
			fmt.Fprintln(p.output, "Paths cannot contain commas or semicolons because Plex uses them as separators.")
			continue
		}
		files = append(files, path)
	}
}

func (p *prompts) confirmOverwrite(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cannot replace %s: expected a regular file", path)
	}
	yes, err := p.yesNo(fmt.Sprintf("%s already exists. Replace it? (y/N): ", path))
	if err != nil {
		return err
	}
	if !yes {
		return fmt.Errorf("setup cancelled; existing files were not changed")
	}
	return nil
}

// writeFile replaces a file with a fully written temporary file. For credentials,
// the temporary file and the final file both have owner-only permissions.
func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".plex-preroll-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func run(input io.Reader, output io.Writer) error {
	p := &prompts{reader: bufio.NewReader(input), input: input, output: output}
	fmt.Fprintln(output, "Plex Preroll Setup")
	fmt.Fprintln(output, "------------------")

	// Read the token before the buffered reader can consume terminal input.
	plexToken, err := p.token()
	if err != nil {
		return err
	}
	var serverURL string
	for {
		serverURL, err = p.line("Enter your PLEX_SERVER_URL (default: http://localhost:32400): ")
		if err != nil {
			return err
		}
		if serverURL == "" {
			serverURL = "http://localhost:32400"
		}
		parsed, parseErr := url.Parse(serverURL)
		if parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == "" {
			break
		}
		fmt.Fprintln(output, "Enter an http:// or https:// URL with a hostname, without user credentials or a fragment.")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("finding your configuration directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}
	configDir = filepath.Join(configDir, "plex-preroll")
	configPath, err := p.savePath("Enter path for PLEX_PREROLL_CFG (config.yaml)", filepath.Join(configDir, "config.yaml"))
	if err != nil {
		return err
	}
	envPath, err := p.savePath("Enter path to save .env file", filepath.Join(configDir, ".env"))
	if err != nil {
		return err
	}
	// A case-only difference may name the same new file on Windows and macOS.
	if strings.EqualFold(configPath, envPath) {
		return fmt.Errorf("config.yaml and .env must use different paths")
	}
	configInfo, configErr := os.Stat(configPath)
	envInfo, envErr := os.Stat(envPath)
	if configErr == nil && envErr == nil && os.SameFile(configInfo, envInfo) {
		return fmt.Errorf("config.yaml and .env must use different files")
	}

	fmt.Fprintln(output, "\nSet up your config.yaml file.")
	config := internal.Config{Seasons: []internal.Season{}, Default: []string{}}
	for {
		add, err := p.yesNo("Add a season? (y/N): ")
		if err != nil {
			return err
		}
		if !add {
			break
		}
		var season internal.Season
		for season.Name == "" {
			season.Name, err = p.line("Enter season name (e.g., halloween, christmas): ")
			if err != nil {
				return err
			}
			if season.Name == "" {
				fmt.Fprintln(output, "Season name cannot be empty.")
			}
		}
		season.From, err = p.date("Enter season start date (MM-DD): ")
		if err != nil {
			return err
		}
		season.To, err = p.date("Enter season end date (MM-DD): ")
		if err != nil {
			return err
		}
		season.Files, err = p.files("Enter a preroll file or directory path for this season (or leave blank to finish): ")
		if err != nil {
			return err
		}
		config.Seasons = append(config.Seasons, season)
	}
	fmt.Fprintln(output, "\nSet up default prerolls, used when no season matches.")
	config.Default, err = p.files("Enter a default preroll file or directory path (or leave blank to finish): ")
	if err != nil {
		return err
	}
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("encoding configuration: %w", err)
	}
	envData, err := godotenv.Marshal(map[string]string{
		"PLEX_TOKEN":       plexToken,
		"PLEX_SERVER_URL":  serverURL,
		"PLEX_PREROLL_CFG": configPath,
	})
	if err != nil {
		return fmt.Errorf("encoding .env: %w", err)
	}
	// Confirm every replacement before writing either file.
	for _, path := range []string{configPath, envPath} {
		if err := p.confirmOverwrite(path); err != nil {
			return err
		}
	}
	if err := writeFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("writing config.yaml: %w", err)
	}
	// Recheck after creating config.yaml: the filesystem may equate names that
	// differ in Unicode normalization or another way beyond case alone.
	configInfo, configErr = os.Stat(configPath)
	envInfo, envErr = os.Stat(envPath)
	if configErr == nil && envErr == nil && os.SameFile(configInfo, envInfo) {
		return fmt.Errorf("config.yaml was saved, but .env resolves to the same file; choose different paths")
	}
	if err := writeFile(envPath, []byte(envData+"\n"), 0600); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}
	fmt.Fprintf(output, "config.yaml saved at %s\n.env saved at %s\n\nSetup complete!\n", configPath, envPath)
	return nil
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "Setup failed:", err)
		os.Exit(1)
	}
}
