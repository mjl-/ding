// Ding is a self-hosted build server for developers.
//
// See the INSTALL.txt file for installation instructions, or run "ding help".
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/mjl-/httpasset"
	"github.com/mjl-/sconf"
)

const (
	databaseVersion = 18
)

var (
	httpFS   = httpasset.Init("assets")
	database *sql.DB

	version       = "dev"
	vcsCommitHash = ""
	vcsTag        = ""
	vcsBranch     = ""
)

var config struct {
	ShowSherpaErrors      bool     `sconf-doc:"If set, returns the full error message for sherpa calls failing with a server error. Otherwise, only returns generic error message."`
	PrintSherpaErrorStack bool     `sconf-doc:"If set, prints error stack for sherpa server errors."`
	Password              string   `sconf-doc:"For login to the web interface. Ding does not have users."`
	DataDir               string   `sconf-doc:"Directory where all data is stored for builds, releases, home directories. In case of isolate builds, this must have a umask 027 and owned by the ding uid/gid. Can be an absolute path, or a path relative to the ding working directory."`
	Database              string   `sconf-doc:"For example: dbname=ding host=localhost port=5432 user=ding password=secret sslmode=disable connect_timeout=3 application_name=ding"`
	GoToolchainDir        string   `sconf:"optional" sconf-doc:"Directory containing Go toolchains, for easy installation of new Go versions. Go toolchains are assumed to be in directories named after their version, e.g. go1.13.8. All names starting with 'go' are assumed to be Go toolchains. Active versions are marked by a symlink named go or go-prev to one of the versioned directories. Ding needs write access to this directory to download new toolchains."`
	Environment           []string `sconf-doc:"List of environment variables in form KEY=VALUE."`
	Notify                struct {
		Name  string `sconf-doc:"Name to use along Email address."`
		Email string `sconf:"optional" sconf-doc:"Address to send build failure notifications to, if Mail.Enabled is set."`
	} `sconf:"optional"`
	BaseURL                string   `sconf-doc:"URL to point to from notifications about failed builds."`
	GithubWebhookSecret    string   `sconf:"optional" sconf-doc:"For github webhook push events, to create a build; configure the same secret in the github repository settings."`
	GiteaWebhookSecret     string   `sconf:"optional" sconf-doc:"For gitea webhooks for builds (like github). With 'Authorization: Bearer <secret>' header for authorization."`
	BitbucketWebhookSecret string   `sconf:"optional" sconf-doc:"Will be part of the URL bitbucket sends its webhook request to, e.g. http://.../bitbucket/<reponame>/<bitbucket-webhook-secret>."`
	Run                    []string `sconf:"optional" sconf-doc:"List of command and arguments to prepend to the command executed, e.g. nice or timeout."`
	IsolateBuilds          struct {
		Enabled  bool   `sconf-doc:"If false, we run all build commands as the user running ding and the settings below do not apply.  If true, we run builds with unique UIDs."`
		UIDStart uint32 `sconf-doc:"We'll use UIDStart + buildID as the unix UID to run the commands under."`
		UIDEnd   uint32 `sconf-doc:"If we reach this UID, we wrap around to UIDStart again."`
		DingUID  uint32 `sconf-doc:"UID ding runs as, used to chown files back before deleting."`
		DingGID  uint32 `sconf-doc:"GID ding runs as, used to run build commands under."`
	}
	Mail struct {
		Enabled      bool `sconf-doc:"If true, emails can be sent for failed builds. If false, the options below do not apply."`
		SMTPTLS      bool
		SMTPPort     int
		SMTPHost     string
		SMTPUsername string `sconf:"optional" sconf-doc:"If not set, no authentication is attempted."`
		SMTPPassword string `sconf:"optional"`
		FromName     string
		FromEmail    string
		ReplyToName  string
		ReplyToEmail string
	}
}

func init() {
	config.DataDir = "data"
	config.BaseURL = "http://localhost:6084"
	config.Mail.SMTPPort = 25
	config.IsolateBuilds.UIDStart = 10000
	config.IsolateBuilds.UIDEnd = 20000
	config.IsolateBuilds.DingUID = 1234
	config.IsolateBuilds.DingGID = 1234
	config.Mail.SMTPHost = "localhost"
	config.Mail.FromName = "ding"
	config.Mail.FromEmail = "ding@example.org"
	config.Mail.ReplyToName = "ding"
	config.Mail.ReplyToEmail = "ding@example.org"
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func initDingDataDir() {
	workdir, err := os.Getwd()
	check(err, "getting current work dir")
	if path.IsAbs(config.DataDir) {
		dingDataDir = path.Clean(config.DataDir)
	} else {
		dingDataDir = path.Join(workdir, config.DataDir)
	}
}

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		log.Fatalf("usage: ding { config | testconfig | help | kick | serve | upgrade | version | license }")
	}
	if len(os.Args) <= 1 {
		flag.Usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "config":
		fmt.Println("# Example config file")
		err := sconf.Describe(os.Stdout, &config)
		check(err, "describe")
	case "testconfig":
		if len(args) != 1 {
			log.Fatalf("usage: ding testconfig config.conf")
		}
		err := sconf.ParseFile(args[0], &config)
		check(err, "parsing file")
		fmt.Println("config OK")
	case "help":
		printFile("/INSTALL.txt")
	case "serve":
		serve(args)
	case "serve-http":
		// undocumented, for unpriviliged http process
		servehttp(args)
	case "upgrade":
		upgrade(args)
	case "kick":
		kick(args)
	case "version":
		fmt.Printf("%s\ndatabase schema version %d\n", version, databaseVersion)
	case "license":
		printFile("/web/LICENSES")
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func printFile(name string) {
	f, err := httpFS.Open(name)
	check(err, "opening file "+name)
	_, err = io.Copy(os.Stdout, f)
	check(err, "copy")
	check(f.Close(), "close")
}
