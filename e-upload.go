// main
package main
import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "os"
    "path"
    "path/filepath"
    "time"
    "github.com/pkg/sftp"
    "golang.org/x/crypto/ssh"
)
// EupConfig mmm
type EupConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
    Path     string `json:"path"`
}
func sshconnect(user, password, host string, port int) (*ssh.Session, error) {
    var (
        auth         []ssh.AuthMethod
        addr         string
        clientConfig *ssh.ClientConfig
        client       *ssh.Client
        session      *ssh.Session
        err          error
    )
    // get auth method
    auth = make([]ssh.AuthMethod, 0)
    auth = append(auth, ssh.Password(password))
    clientConfig = &ssh.ClientConfig{
        User:    user,
        Auth:    auth,
        Timeout: 30 * time.Second,
    }
    // connet to ssh
    addr = fmt.Sprintf("%s:%d", host, port)
    if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
        return nil, err
    }
    // create session
    if session, err = client.NewSession(); err != nil {
        return nil, err
    }
    return session, nil
}
func sftpconnect(user, password, host string, port int) (*sftp.Client, error) {
    var (
        auth         []ssh.AuthMethod
        addr         string
        clientConfig *ssh.ClientConfig
        sshClient    *ssh.Client
        sftpClient   *sftp.Client
        err          error
    )
    // get auth method
    auth = make([]ssh.AuthMethod, 0)
    auth = append(auth, ssh.Password(password))
    clientConfig = &ssh.ClientConfig{
        User:    user,
        Auth:    auth,
        Timeout: 30 * time.Second,
        HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
            return nil
        },
    }
    // connet to ssh
    addr = fmt.Sprintf("%s:%d", host, port)
    if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
        return nil, err
    }
    // create sftp client
    if sftpClient, err = sftp.NewClient(sshClient); err != nil {
        return nil, err
    }
    return sftpClient, nil
}
// single file copy
func scpCopy(sftpClient *sftp.Client, localFilePath string, remotePath string) {
    srcFile, err := os.Open(localFilePath)
    if err != nil {
        fmt.Println("os.Open error : ", localFilePath)
        log.Fatal(err)
    }
    defer srcFile.Close()
    var remoteFileName = path.Base(localFilePath)
    dstFile, err := sftpClient.Create(path.Join(remotePath, remoteFileName))
    if err != nil {
        fmt.Println("sftpClient.Create error : ", path.Join(remotePath, remoteFileName))
        log.Fatal(err)
    }
    defer dstFile.Close()
    ff, err := ioutil.ReadAll(srcFile)
    if err != nil {
        fmt.Println("ReadAll error : ", localFilePath)
        log.Fatal(err)
    }
    dstFile.Write(ff)
    fmt.Println(localFilePath + "  copy file to remote server finished!")
}
func scpCopyDir(sftpClient *sftp.Client, localPath string, remotePath string) {
    localFiles, err := ioutil.ReadDir(localPath)
    if err != nil {
        log.Fatal("read dir list fail ", err)
    }
    for _, backupDir := range localFiles {
        localFilePath := path.Join(localPath, backupDir.Name())
        remoteFilePath := path.Join(remotePath, backupDir.Name())
        if backupDir.IsDir() {
            sftpClient.Mkdir(remoteFilePath)
            scpCopyDir(sftpClient, localFilePath, remoteFilePath)
        } else {
            scpCopy(sftpClient, path.Join(localPath, backupDir.Name()), remotePath)
        }
    }
    // fmt.Println(localPath + "  copy directory to remote server finished!")
}
func main() {
    cmd := os.Args[1]
    fi, _ := os.Stat(cmd)
    cfgPath, err := filepath.Abs("./eup.json")
    data1, err := ioutil.ReadFile(cfgPath)
    var cfg EupConfig
    err = json.Unmarshal(data1, &cfg)
    if err != nil {
        fmt.Println("error in translating,", err.Error())
        return
    }
    sftpClient, _ := sftpconnect(cfg.Username, cfg.Password, cfg.Host, cfg.Port)
    if fi.IsDir() {
        remotePath := path.Join(cfg.Path, cmd)
        sftpClient.Mkdir(remotePath)
        scpCopyDir(sftpClient, cmd, remotePath)
    } else {
        scpCopy(sftpClient, cmd, cfg.Path)
    }
}
