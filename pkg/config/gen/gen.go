package main

import (
	cfg "github.com/conductorone/baton-duo/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("duo", cfg.Config)
}
