package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kardianos/service"
	"github.com/oneworkspacestudio/my-vault/internal/vault"
)

type program struct {
	root   string
	config vault.Config
	cancel context.CancelFunc
	done   chan struct{}
}

func (p *program) Start(s service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	ready := make(chan error, 1)
	go func() {
		e := vault.Run(ctx, p.root, p.config, false, func() { ready <- nil })
		close(p.done)
		select {
		case ready <- e:
		default:
		}
		if e != nil {
			go s.Stop()
		}
	}()
	select {
	case e := <-ready:
		return e
	case <-time.After(120 * time.Second):
		cancel()
		return errors.New("vault service did not become ready")
	}
}
func (p *program) Stop(service.Service) error {
	if p.cancel != nil {
		p.cancel()
		<-p.done
	}
	return nil
}
func serviceFor(root string, p *program) (service.Service, error) {
	exe, e := os.Executable()
	if e != nil {
		return nil, e
	}
	hash := sha256.Sum256([]byte(root))
	return service.New(p, &service.Config{Name: fmt.Sprintf("OneWorkspaceMyVault-%x", hash[:6]), DisplayName: "My Vault", Description: "Independent Vaultwarden vault and encrypted recovery companion", Executable: exe, WorkingDirectory: root, Arguments: []string{"run", "--data", root, "--no-browser"}})
}
func runService(root string, c vault.Config) error {
	s, e := serviceFor(root, &program{root: root, config: c})
	if e != nil {
		return e
	}
	return s.Run()
}
func controlService(root, action string) error {
	s, e := serviceFor(root, &program{})
	if e != nil {
		return e
	}
	switch action {
	case "service-install":
		return s.Install()
	case "service-start":
		return s.Start()
	case "service-stop":
		return s.Stop()
	case "service-uninstall":
		return s.Uninstall()
	}
	return nil
}
