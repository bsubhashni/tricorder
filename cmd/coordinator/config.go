package main

type Config struct {
	Capture CaptureConfig  `yaml:"capture"`
	Agents  []string       `yaml:"agents"`
	Port    int            `yaml:"port"`
	History ResultsHistory `yaml:"history"`
}

type CaptureConfig struct {
	Timeout  int `yaml:"timeout"`
	Period   int `yaml:"period"`
	Interval int `yaml:"interval"`
}

type ResultsHistory struct {
	FileName string `yaml:"file"`
	Period   int    `yaml:"period"`
}
