package main

import (
	"fmt"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"sda-pipeline/internal/broker"
	"sda-pipeline/internal/postgres"
	"sda-pipeline/internal/storage"

	"github.com/spf13/viper"
)

var (
	requiredConfVars = []string{
		"broker.host", "broker.port", "broker.user", "broker.password", "broker.queue", "broker.routingkey",
		"db.host", "db.port", "db.user", "db.password", "db.database",
	}
)

// Config is a parent object for all the different configuration parts
type Config struct {
	Broker   broker.Mqconf
	Postgres postgres.Pgconf
	Archive  storage.Conf
	Inbox    storage.Conf
	Crypt4gh Crypt4gh
}

// Crypt4gh holds c4gh related config info
type Crypt4gh struct {
	KeyPath    string
	Passphrase string
}

// NewConfig initializes and parses the config file and/or environment using
// the viper library.
func NewConfig() *Config {
	parseConfig()

	if viper.GetString("archive.type") == "s3" {
		s3ConfVars := []string{
			"archive.url", "archive.accesskey", "archive.secretkey", "archive.bucket",
		}
		requiredConfVars = append(requiredConfVars, s3ConfVars...)
	} else {
		posixConfVars := []string{
			"archive.location", "archive.uid", "archive.gid",
		}
		requiredConfVars = append(requiredConfVars, posixConfVars...)
	}

	if viper.GetString("inbox.type") == "s3" {
		s3ConfVars := []string{
			"inbox.url", "inbox.accesskey", "inbox.secretkey", "inbox.bucket",
		}
		requiredConfVars = append(requiredConfVars, s3ConfVars...)
	} else {
		posixConfVars := []string{
			"inbox.location",
		}
		requiredConfVars = append(requiredConfVars, posixConfVars...)
	}

	for _, s := range requiredConfVars {
		if !viper.IsSet(s) {
			log.Fatalf("%s not set", s)
		}
	}

	c := &Config{}
	c.readConfig()

	return c
}

func (c *Config) readConfig() {

	// Setup archive
	//nolint:nestif
	if viper.GetString("archive.type") == "s3" {
		s3 := storage.Conf{}
		s3.Type = "s3"
		// All these are required
		s3.URL = viper.GetString("archive.url")
		s3.AccessKey = viper.GetString("archive.accesskey")
		s3.SecretKey = viper.GetString("archive.secretkey")
		s3.Bucket = viper.GetString("archive.bucket")

		if viper.IsSet("archive.port") {
			s3.Port = viper.GetInt("archive.port")
		} else {
			s3.Port = 443
		}

		if viper.IsSet("archive.chunksize") {
			s3.Chunksize = viper.GetInt("archive.chunksize") * 1024 * 1024
		}

		if viper.IsSet("archive.cacert") {
			s3.Cacert = viper.GetString("archive.cacert")
		}

		c.Archive = s3

	} else {
		file := storage.Conf{}
		file.Type = "posix"
		file.Location = viper.GetString("archive.location")
		file.UID = viper.GetInt("archive.uid")
		file.GID = viper.GetInt("archive.gid")

		if viper.IsSet("archive.mode") {
			file.Mode = viper.GetInt("archive.mode")
		} else {
			file.Mode = 2750
		}

		c.Archive = file

	}

	//nolint:nestif
	if viper.GetString("inbox.type") == "s3" {
		s3 := storage.Conf{}
		s3.Type = "s3"
		// All these are required
		s3.URL = viper.GetString("inbox.url")
		s3.AccessKey = viper.GetString("inbox.accesskey")
		s3.SecretKey = viper.GetString("arcinboxhive.secretkey")
		s3.Bucket = viper.GetString("inbox.bucket")

		if viper.IsSet("inbox.port") {
			s3.Port = viper.GetInt("inbox.port")
		} else {
			s3.Port = 443
		}

		if viper.IsSet("inbox.chunksize") {
			s3.Chunksize = viper.GetInt("inbox.chunksize") * 1024 * 1024
		}

		if viper.IsSet("inbox.cacert") {
			s3.Cacert = viper.GetString("inbox.cacert")
		}

		c.Archive = s3

	} else {
		file := storage.Conf{}
		file.Type = "posix"
		file.Location = viper.GetString("inbox.location")
		file.UID = viper.GetInt("inbox.uid")
		file.GID = viper.GetInt("inbox.gid")

		if viper.IsSet("inbox.mode") {
			file.Mode = viper.GetInt("inbox.mode")
		} else {
			file.Mode = 2660
		}

		c.Inbox = file

	}

	// Setup broker
	b := broker.Mqconf{}

	b.Host = viper.GetString("broker.host")
	b.Port = viper.GetInt("broker.port")
	b.User = viper.GetString("broker.user")
	b.Password = viper.GetString("broker.password")
	b.RoutingKey = viper.GetString("broker.routingkey")
	b.Queue = viper.GetString("broker.queue")
	b.ServerName = viper.GetString("broker.serverName")

	if viper.IsSet("broker.routingerror") {
		b.RoutingError = viper.GetString("broker.routingerror")
	}
	if viper.IsSet("broker.vhost") {
		if strings.HasPrefix(viper.GetString("broker.vhost"), "/") {
			b.Vhost = viper.GetString("broker.vhost")
		} else {
			b.Vhost = "/" + viper.GetString("broker.vhost")
		}
	} else {
		b.Vhost = "/"
	}

	if viper.IsSet("broker.ssl") {
		b.Ssl = viper.GetBool("broker.ssl")
	}
	if viper.IsSet("broker.verifyPeer") {
		b.VerifyPeer = viper.GetBool("broker.verifyPeer")
		// Since verifyPeer is specified, these are required.
		if !(viper.IsSet("broker.clientCert") && viper.IsSet("broker.clientKey")) {
			log.Fatalln("when broker.verifyPeer is set both broker.clientCert and broker.clientKey is needed")
		}
		b.ClientCert = viper.GetString("broker.clientCert")
		b.ClientKey = viper.GetString("broker.clientKey")
	}
	if viper.IsSet("broker.cacert") {
		b.Cacert = viper.GetString("broker.cacert")
	}

	c.Broker = b

	db := postgres.Pgconf{}

	// All these are required
	db.Host = viper.GetString("db.host")
	db.Port = viper.GetInt("db.port")
	db.User = viper.GetString("db.user")
	db.Password = viper.GetString("db.password")
	db.Database = viper.GetString("db.database")
	db.SslMode = viper.GetString("db.sslmode")

	// Optional settings
	if viper.IsSet("db.sslMode") && viper.GetString("db.sslMode") == "verify-full" {
		db.SslMode = viper.GetString("db.sslMode")
		// Since verify-full is specified, these are required.
		if !(viper.IsSet("db.clientCert") && viper.IsSet("db.clientKey")) {
			panic(fmt.Errorf("when db.sslMode is set to verify-full both db.clientCert and db.clientKey are needed"))
		}
		db.ClientCert = viper.GetString("db.clientCert")
		db.ClientKey = viper.GetString("db.clientKey")
	}
	if viper.IsSet("db.cacert") {
		db.Cacert = viper.GetString("db.cacert")
	}

	c.Postgres = db

	if viper.IsSet("log.level") {
		switch strings.ToLower(viper.GetString("log.level")) {
		case "info":
			log.SetLevel(log.InfoLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		}
		log.Printf("Setting loglevel to %s", strings.ToLower(viper.GetString("log.level")))
	}

	c.Crypt4gh.KeyPath = viper.GetString("c4gh.filepath")
	c.Crypt4gh.Passphrase = viper.GetString("c4gh.passphrase")

}

func parseConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigType("yaml")
	if viper.IsSet("configPath") {
		cp := viper.GetString("configPath")
		ss := strings.Split(strings.TrimLeft(cp, "/"), "/")
		viper.AddConfigPath(path.Join(ss...))
	}
	if viper.IsSet("configFile") {
		viper.SetConfigFile(viper.GetString("configFile"))
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Infoln("No config file found, using ENVs only")
		} else {
			log.Fatalf("fatal error config file: %s", err)
		}
	}
}
