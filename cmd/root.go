/*
Copyright © 2024 huangnauh <huanglibo2010@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/moby/sys/mountinfo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func errorExit(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

func isRclone(fsType string) bool {
	return strings.Contains(fsType, "rclone")
}

func isSubFolder(base, sub string) (bool, error) {
	relPath, err := filepath.Rel(base, sub)
	if err != nil {
		return false, err
	}
	return !filepath.IsAbs(relPath) && !strings.Contains(relPath, ".."), nil
}

type Remote struct {
	Name       string `json:"name" yaml:"name"`
	Bucket     string `json:"bucket" yaml:"bucket"`
	Mountpoint string `json:"mountpoint" yaml:"mountpoint"`
}

func remotePath(path string, remotes []Remote) (string, bool) {
	for _, remote := range remotes {
		ok, _ := isSubFolder(remote.Mountpoint, path)
		if ok {
			p := path[len(remote.Mountpoint):]
			p = strings.TrimPrefix(p, "/")
			return strings.TrimSuffix(fmt.Sprintf("%s:%s/%s", remote.Name, remote.Bucket, p), "/"), true
		}
	}
	return path, false
}

var version string = "v0.3"

func copyRemote(source, dest, d string, remotes []Remote) {
	s, err := filepath.Abs(source)
	if err != nil {
		errorExit("stat %s: %s", source, err.Error())
	}
	if s == d {
		errorExit("Nothing to do as '%s' and '%s' are the same", source, dest)
	}

	sRemote, sOk := remotePath(s, remotes)
	dRemote, dOk := remotePath(d, remotes)
	if !sOk && !dOk {
		errorExit("Recommended to use the 'cp' command")
	}
	if dOk {
		prompt := promptui.Prompt{
			Label:     "Uploading will not be immediately visible in the file system; Would you like to continue?",
			IsConfirm: true,
			Default:   "n",
			Validate: func(s string) error {
				if len(s) <= 1 && strings.Contains("yYnN", s) {
					return nil
				}
				return fmt.Errorf("invalid input")
			},
		}
		result, err := prompt.Run()
		if err != nil {
			errorExit(err.Error())
		}
		if strings.Contains("nN", result) {
			errorExit("Abort")
		}
	}
	sinfo, err := os.Stat(s)
	if err != nil {
		errorExit("stat %s: %s", source, err.Error())
	} else if sinfo.IsDir() {
		sRemote += "/"
	} else {
		if strings.HasSuffix(source, "/") {
			errorExit("'%s' is a file, not a directory.", source)
		}
	}
	dfolder := false
	dinfo, err := os.Stat(d)
	if os.IsNotExist(err) {
		if strings.HasSuffix(dest, "/") {
			err = os.MkdirAll(d, 0755)
			if err != nil {
				errorExit(err.Error())
			}
			dfolder = true
		}
	} else if err != nil {
		errorExit("stat %s: %s", dest, err.Error())
	} else {
		if !dinfo.IsDir() && sinfo.IsDir() {
			errorExit("cannot overwrite non-directory '%s' with directory '%s'", dest, source)
		}
		if !dinfo.IsDir() && strings.HasSuffix(dest, "/") {
			errorExit("'%s' is a file, not a directory.", dest)
		}
	}
	if dfolder || (dinfo != nil && dinfo.IsDir()) {
		dRemote += "/"
		if !sinfo.IsDir() {
			dRemoteFile := filepath.Join(d, filepath.Base(s))
			dsub, err := os.Stat(dRemoteFile)
			if err == nil && dsub.IsDir() {
				errorExit("cannot overwrite directory '%s' with file '%s'", dRemoteFile, source)
			}
			dRemote += filepath.Base(s)
		}
	}
	a := []string{"copyto", sRemote, dRemote, fmt.Sprintf("--transfers=%d", parallel),
		fmt.Sprintf("--checkers=%d", parallel), "-P"}
	c := exec.Command("rclone", a...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	logrus.Debugf("'%s' => '%s'", sRemote, dRemote)
	err = c.Start()
	if err != nil {
		errorExit("%s", err)
	}
	err = c.Wait()
	if err != nil {
		errorExit("%s", err)
	}
}

func GetMounts() []Remote {
	mounts, err := mountinfo.GetMounts(func(mount *mountinfo.Info) (skip, stop bool) {
		ok := isRclone(mount.FSType)
		if ok {
			logrus.Debugf("remote mountpoint %s\n", mount.Mountpoint)
			return false, false
		}
		return true, false
	})
	if err != nil {
		errorExit("mount info: %s", err.Error())
	}
	remotes := []Remote{}
	for _, mount := range mounts {
		ss := strings.Split(mount.Source, ":")
		if len(ss) != 2 {
			continue
		}
		name := ss[0]
		bucket := ss[1]
		n := strings.Split(name, "{")
		if len(n) > 2 {
			continue
		}
		if len(n) == 2 {
			name = n[0]
		}
		remotes = append(remotes, Remote{
			Name:       name,
			Bucket:     bucket,
			Mountpoint: mount.Mountpoint,
		})
	}

	cfg, err := ReadConfig("")
	if err != nil {
		errorExit("read config: %s", err.Error())
	}
	if cfg == nil {
		return remotes
	}
	remotes = append(cfg.Remotes, remotes...)
	return remotes
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "rcp",
	Short:   "Copy SOURCE to DEST",
	Long:    `Copy SOURCE to DEST`,
	Example: "rcp SOURCE DEST",
	Version: version,
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if parallel > 12 {
			errorExit("The maximum number of parallel transfers is 12.")
		} else if parallel < 1 {
			errorExit("The minimum number of parallel transfers is 1.")
		}
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}

		remotes := GetMounts()
		if save {
			cfg := &Config{
				Remotes: remotes,
			}
			err := cfg.Write()
			if err != nil {
				errorExit(err.Error())
			}
			return
		}
		if len(remotes) == 0 {
			errorExit("Recommended to use the 'cp' command")
		}

		dest := args[len(args)-1]
		d, err := filepath.Abs(dest)
		if err != nil {
			errorExit("stat %s: %s", dest, err.Error())
		}
		for _, arg := range args[:len(args)-1] {
			copyRemote(arg, dest, d, remotes)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var parallel int
var verbose bool
var save bool

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rcp.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	// rootCmd.Flags().BoolVar(&save, "save", false, "save config")
	rootCmd.Flags().IntVar(&parallel, "parallel", 6, "The number of file transfers to run in parallel.")
}
