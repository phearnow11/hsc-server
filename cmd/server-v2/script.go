package main

import (
	"log"
	"os/exec"
	"path/filepath"
	"pigdata/datadog/processingServer/pkg/utils"
	"time"
)

var (
	scripts_dir_name         = "cronjob-scripts"
	scripts_config_file_name = "cronjob-config.yaml"
	scripts_config           = Script_Config{}
	_interval                time.Duration
	script_status            = false
)

func init() {
	log.Printf("Scripts: Loading config file: %s", utils.ReadYAMLFile(scripts_config_file_name, &scripts_config))
	if scripts_config.Interval == "" {
		log.Printf("Scripts: Error: Interval is empty (Config loading error)")
		return
	}
	interval, err := time.ParseDuration(scripts_config.Interval)
	if err != nil {
		log.Printf("Scripts: Error parsing interval time from config file: %s", err)
		return
	} else {
		_interval = interval
	}
	script_status = true
}

type Script struct {
	Name string   `yaml:"name"`
	Args []string `yaml:"args"`
}

type Script_Config struct {
	Interval string   `yaml:"interval"`
	Scripts  []Script `yaml:"scripts"`
}

func runScripts(scripts []Script) {
	scriptDir := "./" + scripts_dir_name
	for _, script := range scripts {
		scriptPath := filepath.Join(scriptDir, script.Name)
		log.Printf("Scripts: Executing script: %s with args: %v", script.Name, script.Args)

		var cmd *exec.Cmd
		switch filepath.Ext(script.Name) {
		case ".sh":
			cmd = exec.Command("/bin/bash", append([]string{scriptPath}, script.Args...)...)
		case ".py":
			cmd = exec.Command("python", append([]string{scriptPath}, script.Args...)...)
		case ".go":
			cmd = exec.Command("go", append([]string{"run", scriptPath}, script.Args...)...)
		default:
			log.Printf("Scripts: Skipping unsupported script type: %s", script.Name)
			continue
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Scripts: Error executing %s: %v\nOutput: %s", script.Name, err, output)
		} else {
			log.Printf("Scripts: Output of %s:\n%s", script.Name, output)
		}
	}
}
