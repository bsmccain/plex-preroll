package internal

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Season struct {
	Name  string   `yaml:"name"`
	From  string   `yaml:"from"`
	To    string   `yaml:"to"`
	Files []string `yaml:"files"`
}

type Config struct {
	Seasons []Season `yaml:"seasons"`
	Default []string `yaml:"default"`
}

func LoadConfig(path string) (*Config, error) {
	yamlFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()

	var config *Config
	decoder := yaml.NewDecoder(yamlFile)
	decoder.KnownFields(true)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	if config == nil || (config.Seasons == nil && config.Default == nil) {
		return nil, fmt.Errorf("config must define seasons or default; use default: [] to intentionally clear prerolls")
	}
	var extraDocument any
	if err := decoder.Decode(&extraDocument); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("invalid trailing YAML: %w", err)
		}
		return nil, fmt.Errorf("config must contain a single YAML document")
	}

	for i := range config.Seasons {
		if config.Seasons[i].Files == nil {
			return nil, fmt.Errorf("season %q must define files; use files: [] to intentionally clear prerolls", config.Seasons[i].Name)
		}
		from, err := NormalizeMMDD(config.Seasons[i].From)
		if err != nil {
			return nil, fmt.Errorf("season %q has invalid 'from' date format: %v", config.Seasons[i].Name, err)
		}
		to, err := NormalizeMMDD(config.Seasons[i].To)
		if err != nil {
			return nil, fmt.Errorf("season %q has invalid 'to' date format: %v", config.Seasons[i].Name, err)
		}
		config.Seasons[i].From = from
		config.Seasons[i].To = to
	}

	return config, nil
}

// NormalizeMMDD validates a recurring calendar date and pads its month and day.
// February 29 is valid so seasons can include leap days.
func NormalizeMMDD(date string) (string, error) {
	parts := strings.Split(date, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid date format: %v", date)
	}
	month, err1 := strconv.Atoi(parts[0])
	day, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return "", fmt.Errorf("invalid date: %v", date)
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return "", fmt.Errorf("invalid date: %v", date)
	}
	calendarDate := time.Date(2000, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if int(calendarDate.Month()) != month || calendarDate.Day() != day {
		return "", fmt.Errorf("invalid calendar date: %v", date)
	}
	return fmt.Sprintf("%02d-%02d", month, day), nil
}

func (c *Config) GetPrerolls(today time.Time) []string {
	prerolls := []string{}
	matched := false
	todayMMDD := today.Format("01-02")
	for _, season := range c.Seasons {
		if season.From <= season.To {
			if todayMMDD >= season.From && todayMMDD <= season.To {
				matched = true
				prerolls = append(prerolls, season.Files...)
			}
		} else {
			if todayMMDD >= season.From || todayMMDD <= season.To {
				matched = true
				prerolls = append(prerolls, season.Files...)
			}
		}
	}

	if !matched {
		prerolls = append(prerolls, c.Default...)
	}

	sort.Strings(prerolls)
	return prerolls
}
