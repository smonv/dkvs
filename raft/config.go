package raft

import (
	"log"
	"os"
)

// Config provide any necessary config for Raft node
type Config struct {
	HeartbeatTimeout int64
	ElectionTimeout  int64
	Logger           *log.Logger
}

// DefaultConfig return default config for Raft node
func DefaultConfig() *Config {
	return &Config{
		HeartbeatTimeout: 50,
		ElectionTimeout:  150,
		Logger:           log.New(os.Stdout, "", log.LstdFlags),
	}
}
