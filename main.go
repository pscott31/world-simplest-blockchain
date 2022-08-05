package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/viper"

	abciclient "github.com/tendermint/tendermint/abci/client"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmconfig "github.com/tendermint/tendermint/config"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmservice "github.com/tendermint/tendermint/libs/service"
	tmnode "github.com/tendermint/tendermint/node"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "./tmhome/config/config.toml", "Path to config.toml")
}

func main() {
	app := NewKVStoreApplication()

	flag.Parse()

	node, err := newTendermint(app, configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	node.Start()
	defer func() {
		node.Stop()
		node.Wait()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func newTendermint(app abcitypes.Application, configFile string) (tmservice.Service, error) {
	// read config
	config := tmconfig.DefaultValidatorConfig()
	config.SetRoot(filepath.Dir(filepath.Dir(configFile)))

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("viper failed to read config file: %w", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("viper failed to unmarshal config: %w", err)
	}
	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	// create logger
	logger, err := tmlog.NewDefaultLogger(tmlog.LogFormatPlain, config.LogLevel, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// create node
	node, err := tmnode.New(
		config,
		logger,
		abciclient.NewLocalCreator(app),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Tendermint node: %w", err)
	}

	return node, nil
}
