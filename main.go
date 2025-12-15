package main

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"github.com/studio-b12/gowebdav"
	"github.com/urfave/cli/v3"
	"os"
	"path/filepath"
)

func main() {
	initConfig()
	initCommand()
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/smallsync")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("err")
		fmt.Println(err == viper.ConfigFileNotFoundError{})
		var configPath string
		if home, err := os.UserHomeDir(); err != nil {
			fmt.Println("cannot create config file at $HOME/.config/smallsync/")
			configPath = "."
		} else {
			configPath = filepath.Join(home, "/.config/smallsync/")
		}
		configFilepath := filepath.Join(configPath, "config.yaml")
		if err := os.MkdirAll(configPath, 0755); err != nil {
			panic("create dir error")
		}
		file, err := os.Create(configFilepath)
		if err != nil {
			panic(err)
		}
		viper.SetDefault("remote.type", "webdav")
		viper.WriteConfigAs(configFilepath)
		file.Close()
	}
}

func initCommand() {
	cmd := &cli.Command{
		Name:  "smallsync",
		Usage: "sync single file to server",
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "test",
				Usage: "test the server",
				Action: func(context.Context, *cli.Command) error {
					fmt.Println("test server:", cmdTestServer())
					return nil
				},
			},

			&cli.Command{
				Name:  "server",
				Usage: "config the remote server",
				Action: func(context.Context, *cli.Command) error {
					cmdSetupServer()
					return nil
				},
			},

			&cli.Command{
				Name:  "add",
				Aliases:  []string{"a"},				
				Usage: "add a remote/local entry for sync",
				Action: func(context.Context, *cli.Command) error {
					cmdAddEntry()
					return nil
				},
			},

			&cli.Command{
				Name:  "list",
				Aliases:  []string{"l"},
				Usage: "list all the entries",
				Action: func(context.Context, *cli.Command) error {
					cmdListEntry()
					return nil
				},
			},

			&cli.Command{
				Name: "upload",
				Aliases: []string{"u","up"},
				Usage: "upload entries",
				Arguments: []cli.Argument {
					&cli.StringArg{
						Name: "entry",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					uploadOneEntry(cmd.StringArg("entry"))
					return nil
				},
			},
			
			&cli.Command{
				Name: "download",
				Aliases: []string{"d","down"},
				Usage: "download entries",
				Arguments: []cli.Argument {
					&cli.StringArg{
						Name: "entry",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// TODO check entry name
					downloadOneEntry(cmd.StringArg("entry"))
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}

func inputContext(prompt string, result *string) bool {
	fmt.Print(prompt)
	_, err := fmt.Scanln(result)
	if err != nil {
		fmt.Println("Input error", err)
		return false
	}
	return true
}

func cmdSetupServer() {
	var input string
	for !inputContext("Input Server path:", &input) {
	}
	viper.Set("remote.webdav.serverPath", input)

	for !inputContext("Input Server username:", &input) {
	}
	viper.Set("remote.webdav.username", input)

	for !inputContext("Input Server password:", &input) {
	}
	viper.Set("remote.webdav.password", input)

	if err := viper.WriteConfig(); err != nil {
		panic(err)
	}
	fmt.Println("\nServer config saved!")
}

func cmdTestServer() bool {
	c := gowebdav.NewClient(
		viper.GetString("remote.webdav.serverPath"),
		viper.GetString("remote.webdav.username"),
		viper.GetString("remote.webdav.password"))
	if err := c.Connect(); err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println("Test: Server connected successfully!")

	info, _ := c.Stat("/")
	fmt.Println(info)

	return true
}

func cmdAddEntry() {
	var input, name string
	for !inputContext("Input entry name:", &name) {
	}

	for !inputContext("Input local filepath:", &input) {
	}
	viper.Set("entry."+name+".local", input)

	for !inputContext("Input Server filepath:", &input) {
	}
	viper.Set("entry."+name+".remote", input)

	if err := viper.WriteConfig(); err != nil {
		panic(err)
	}
	fmt.Println("\nEntry", name, "saved!")
}

func cmdListEntry() {
	entrySub := viper.Sub("entry")
	if entrySub == nil {
		fmt.Println("no entries, add one now!")
		cmdAddEntry()
		return
	}

	entries := entrySub.AllSettings()
	for name, config := range entries {
		fmt.Print("Entry[", name,"]: ")
		if pair, ok := config.(map[string]any); ok {
			fmt.Println(pair["local"], "==>", pair["remote"])
		}
	}

}

func downloadOneEntry(entryName string) bool {
	// check local file exists
	local := viper.GetString("entry." + entryName + ".local")
	remote := viper.GetString("entry." + entryName + ".remote")

	c := gowebdav.NewClient(
		viper.GetString("remote.webdav.serverPath"),
		viper.GetString("remote.webdav.username"),
		viper.GetString("remote.webdav.password"))
	if err := c.Connect(); err != nil {
		fmt.Println(err)
		return false
	}

	if info, err := c.Stat(remote); info == nil || err != nil {
		fmt.Println("remote file not exist")
		return false
	}

	fmt.Print("will overwrite local file [", local, "], are you OK? [Y]")
	var response string
	fmt.Scanln(&response)
	if response == "y" || response == "Y" || response == "" {
		// do download
		bytes, _ := c.Read(remote)
		os.WriteFile(local, bytes, 0644)
		return true
	} else {
		fmt.Println("User aborted.")
		return false
	}
	return false
}

func uploadOneEntry(entryName string) bool {
	// check local file exists
	local := viper.GetString("entry." + entryName + ".local")
	remote := viper.GetString("entry." + entryName + ".remote")
	if info, err := os.Stat(local); err != nil || info == nil {
		return false
	}
	fmt.Print("will overwrite the file [", remote, "] on remote server, are you OK? [Y]")
	var response string
	fmt.Scanln(&response)
	if response == "y" || response == "Y" || response == "" {
		// do upload
		fmt.Println("do upload")
		bytes, _ := os.ReadFile(local)
		
		c := gowebdav.NewClient(
			viper.GetString("remote.webdav.serverPath"),
			viper.GetString("remote.webdav.username"),
			viper.GetString("remote.webdav.password"))
		if err := c.Connect(); err != nil {
			fmt.Println(err)
			return false
		}

		c.Write(remote, bytes, 0644)
		return true
	} else {
		fmt.Println("User aborted.")
		return false
	}
	return false
}
