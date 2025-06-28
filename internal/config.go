package internal

import (
	"fmt"
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

	var config Config
	err = yaml.NewDecoder(yamlFile).Decode(&config)
	if err != nil {
		return nil, err
	}

	for i := range config.Seasons {
		from, err := normalizeMMDD(config.Seasons[i].From)
		if err != nil {
			return nil, fmt.Errorf("season %q has invalid 'from' date format: %v", config.Seasons[i].Name, err)
		}
		to, err := normalizeMMDD(config.Seasons[i].To)
		if err != nil {
			return nil, fmt.Errorf("season %q has invalid 'to' date format: %v", config.Seasons[i].Name, err)
		}
		config.Seasons[i].From = from
		config.Seasons[i].To = to
	}

	return &config, nil
}

func normalizeMMDD(date string) (string, error) {
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
	return fmt.Sprintf("%02d-%02d", month, day), nil
}

func (c *Config) GetPrerolls(today time.Time) []string {
	prerolls := []string{}
	todayMMDD := today.Format("01-02")
	for _, season := range c.Seasons {
		if season.From < season.To {
			if todayMMDD >= season.From && todayMMDD <= season.To {
				prerolls = append(prerolls, season.Files...)
			}
		} else {
			if todayMMDD >= season.From || todayMMDD <= season.To {
				prerolls = append(prerolls, season.Files...)
			}
		}
	}

	if len(prerolls) == 0 {
		prerolls = c.Default
	}

	sort.Strings(prerolls)
	return prerolls
}
