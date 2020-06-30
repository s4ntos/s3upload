package main

import (
      //"fmt"
      "time"
      "bytes"
      "log"
      "net/http"
      "os"
      "io"
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
      "github.com/aws/aws-sdk-go/aws/credentials"
      "github.com/aws/aws-sdk-go/service/s3"
)

type config struct {
   Region string
   Profile string
   Debug bool
   Access_key string
   Secret_key string
   Bucket string
   ACL string
   EncryptionKey string
   Folder string
   Archive bool
   Remove bool
   Locations []string
   Filter string
   Upload_existing bool
   RunOnce bool
}

var (
    conf    config
    Info    *log.Logger
    Error   *log.Logger
)

func Init(
    infoHandle io.Writer,
    errorHandle io.Writer) {

    Info = log.New(infoHandle,
        "INFO: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Error = log.New(errorHandle,
        "ERROR: ",
        log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
      /// Start - Read all configurations
      var file string
      Init(os.Stdout, os.Stderr)
      if len(os.Args) > 1 {
          var extension = filepath.Ext(os.Args[1])
          file = os.Args[1][0:len(os.Args[1])-len(extension)]
          viper.SetConfigName(file)
      } else {
          viper.SetConfigName("s3upload")
      }
      Info.Println("Version 1.0")
      Info.Println("Reading configuration file")
      viper.SetConfigType("yml")
      viper.SetDefault("Debug", false)
      viper.SetDefault("Profile", "default")
      viper.AddConfigPath("conf")   // path to look for the config file in
      viper.AddConfigPath(".")         // optionally look for config in the working directory
      if err := viper.ReadInConfig(); err != nil {
        Error.Fatalf("config file problem. No config file on locations ('.' or 'conf') or error on file", err)
      }
      err := viper.Unmarshal(&conf)
      if err != nil {
        Error.Fatalf("unable to decode into struct, %v", err)
      }
      // Start watching configuration files to perform reloads - will be required for AWS authentication changes
      viper.WatchConfig()
      viper.OnConfigChange(func(e fsnotify.Event) {
        Info.Print("Config file changed :", e.Name)
        viper.Unmarshal(&conf)
        Info.Println ("Updates and reloads done")
      })
      if (conf.Debug) { log.Println("Configurations loaded", conf)}
      // End - Read All configurations
      Info.Println("Preparing AWS session")
      // Create a single AWS session (we can re use this if we're uploading many files)
      s, err := session.NewSessionWithOptions( session.Options{
        Profile: conf.Profile,
        Config: aws.Config{
          Region: aws.String(conf.Region),
          Credentials: credentials.NewStaticCredentials(conf.Access_key, conf.Secret_key, ""),
        },
      })
      if err != nil {
        Error.Fatal(err)
      }
      _ = s
      Info.Println("Preparing filter to watch the directory with the following filter")
      // Filter what to wach for
      r := regexp.MustCompile(conf.Filter)
      // Prepare to watch Files
      watcher, err := fsnotify.NewWatcher()
      if err != nil {
        Error.Fatal(err)
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
                Info.Println("File created time to upload:", event.Name)
                go AddFileToS3(s, conf.Bucket, event.Name, conf.Folder)
             }
          case err, ok := <-watcher.Errors:
             if !ok {
                return
             }
             Error.Println(err)
          }
        }
      }()

      // Setting up all monitors and uploading files if required
      for _, Location := range conf.Locations {
        if conf.Upload_existing {
          Info.Println("Listing files for directory: " + Location)
          fileList, err := listFiles(Location, r)
          if err != nil {
             Error.Fatal("Problems Listing files on Directory:",err)
          }
          for path, _ := range fileList {
             Info.Printf("Uploading files under the criteria " + path)
             AddFileToS3(s, conf.Bucket, path, conf.Folder)
          }
        }
        if ! conf.RunOnce {
          Info.Println("Monitor directory :", Location)
          err = watcher.Add(Location)
          if err != nil {
             Error.Fatal(err)
          }
        }
      }
      if conf.RunOnce {close(done)}
      <-done
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, bucket string, fileDir string, folder string)  {
      currentTime := time.Now()
      // Open the file for use
      file, err := os.Open(fileDir)
      if err != nil {
        Error.Printf("Unable to open file : %s - %s", fileDir, err)
        return
      }
      defer file.Close()

      // Get file size and read the file content into a buffer
      fileInfo, _ := file.Stat()
      var size int64 = fileInfo.Size()
      buffer := make([]byte, size)
      file.Read(buffer)
      Info.Printf("Uploading %s to %s/%s", fileDir,bucket,folder + currentTime.Format("2006/01/02/") + fileInfo.Name())
      input := &s3.PutObjectInput{
                Bucket:               aws.String(bucket),
                Key:                  aws.String(folder + currentTime.Format("2006/01/02/") + fileInfo.Name()),
                Body:                 bytes.NewReader(buffer),
                ContentLength:        aws.Int64(size),
                ContentType:          aws.String(http.DetectContentType(buffer)),
                ContentDisposition:   aws.String("attachment"),
        }
      if conf.ACL != "" {
           input.SetACL(conf.ACL)
      }
      // Lets take care of encryption
      if conf.EncryptionKey == "AES256" {
             input.SetServerSideEncryption("AES256")
      } else {
             if conf.EncryptionKey != "" {
                 if (conf.Debug) { Info.Println("Key to be used:" + conf.EncryptionKey)}
                 input.SetServerSideEncryption("aws:kms")
                 input.SetSSEKMSKeyId(conf.EncryptionKey)
           }
           // else Don't encrypt
        }
      _, err = s3.New(s).PutObject(input)
      if err != nil {
        Error.Printf("Unable to upload file : %s - %s", fileDir, err)
      } else {
          if conf.Archive {
              if conf.Remove {
          Info.Print(".. Removing .. ")
          os.Remove(fileDir)
          Info.Println("Done")
        } else {
          Info.Print(".. Moving .. ")
          os.Rename(fileDir, fileDir + ".done")
          Info.Println("Done")
        }
          }
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
