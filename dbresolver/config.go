package dbresolver

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

/*
DatabaseConfig links API keys to database objects in a map data structure.
*/
type DatabaseConfig map[string]map[string]string

// Returns a struct of DBDrivers with each having a Database and Driver field.
func (dbconfig DatabaseConfig) DatabaseDrivers() []DBDriver {
	databaseNames := make([]DBDriver, 0, len(dbconfig))
	for _, dbmap := range dbconfig {
		driver, exists := dbmap["driver"]
		if !exists {
			panic("driver not specified in configuration file")
		}

		database, exists := dbmap["database"]
		if !exists {
			panic("database key not specified in configuration file")
		}

		databaseNames = append(databaseNames, DBDriver{
			Driver:   Driver(driver),
			Database: database,
		})

	}
	return databaseNames
}

/*
DatabaseConfigFromYAML parses a YAML-formatted string and returns a DatabaseConfig.
*/
func ConfigFromYAMLString(yamlStr string) (DatabaseConfig, error) {
	var config DatabaseConfig
	err := yaml.Unmarshal([]byte(yamlStr), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}
	return config, nil
}

/*
DatabaseConfigFromYAML parses a yaml file and returns a DatabaseConfig.
*/
func ConfigFromYAMLFile(filename string) (DatabaseConfig, error) {
	var config DatabaseConfig
	file, err := os.Open(filename)
	if err != nil {
		return DatabaseConfig{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return DatabaseConfig{}, err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}
	return config, nil
}
