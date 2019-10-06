package main

import (
	"fmt"
	//"time"
	"bytes"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"path/filepath"
	"regexp"
	// config file
	"github.com/spf13/viper"
	// file watcher
	"github.com/fsnotify/fsnotify"
	// AWS SDK 
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type config struct {
   Region string
   Access_key string
   Secret_key string
   Bucket string
   Locations []string
   Filter string
   Upload_existing bool
}

func main() {
	/// Start - Read all configurations
	var conf config
	log.Println("Reading configuration file")
	viper.SetConfigName("wmrupload")
	viper.SetConfigType("yml")
	viper.AddConfigPath("conf")   // path to look for the config file in
	viper.AddConfigPath(".")         // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("config file problem. No config file on locations ('.' or 'conf') or error on file", err)
	}
	err := viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}
	// Start watching configuration files to perform reloads - will be required for AWS authentication changes
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Print("Config file changed :", e.Name)
		viper.Unmarshal(&conf)
		fmt.Println(conf)
		log.Println ("Updates and reloads done")
	})
	// log.Println("Configurations loaded", conf)
	// End - Read All configurations 
	log.Println("Preparing AWS session")
	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(conf.Region)})
	if err != nil {
		log.Fatal(err)
	}
	_ = s
	log.Println("Preparing filter  to watch the directory with the following filter")
	// Filter what to wach for 
	r := regexp.MustCompile(conf.Filter)
	// Prepare to watch Files 
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// setting up watcher function
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ( event.Op == fsnotify.Create && r.Match([]byte(event.Name)) )  {
					log.Println("File created time to upload:", event.Name)
					go AddFileToS3(s, conf.Bucket, event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Setting up all monitors and uploading files if required 
	for _, Location := range conf.Locations {
		if conf.Upload_existing {
			log.Println("Listing files for directory: " + Location)
			fileList, err := listFiles(Location, r) 
			if err != nil {
				log.Fatal("Problems Listing files on Directory:",err)
			}
			for path, _ := range fileList {
				log.Printf("Uploading files under the criteria " + path)
				AddFileToS3(s, conf.Bucket, path)
			}
		}
		log.Println("Monitor directory :", Location)
		err = watcher.Add(Location)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, bucket string, fileDir string)  {

	// Open the file for use
	file, err := os.Open(fileDir)
	if err != nil {
		log.Fatalf("Unable to open file : %s - %s", fileDir, err)
		return 
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)
	log.Printf("Uploading %s", fileDir)

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(fileInfo.Name()),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		log.Fatalf("ERRROR Unable to upload file : %s - %s", fileDir, err)
	} else {
		log.Println("Done")
	}
}

func listFiles(name string, r *regexp.Regexp) (map[string]os.FileInfo, error) {
	fileList := make(map[string]os.FileInfo)

	// Make sure name exists.
	stat, err := os.Stat(name)
	if err != nil {
		return nil, err
	}

	// If it's not a directory, just return.
	if !stat.IsDir() {
		return fileList, nil
	}

	// It's a directory.
	fInfoList, err := ioutil.ReadDir(name)
	if err != nil {
		return nil, err
	}
	// Add all of the files in the directory to the file list as long
	// as they are on the filter
	for _, fInfo := range fInfoList {
		path := filepath.Join(name, fInfo.Name())
		if r.Match([]byte(path)) {
			fileList[path] = fInfo
		}
	}
	return fileList, nil
}