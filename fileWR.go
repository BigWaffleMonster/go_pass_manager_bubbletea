package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/google/uuid"
)

// Struct for app config
type AppConfig struct {
	DBsFolder string `koanf:"dbs_folder"`
}

// Структуры для парсинга JSON
type PasswordFile struct {
	Header   Header   `json:"header"`
	Database Database `json:"database"`
}

type Header struct {
	Crypto Crypto `json:"crypto"`
}

type Crypto struct {
	Cipher      string `json:"cipher"`
	Compression string `json:"compression"`
}

type Database struct {
	Meta    Meta    `json:"meta"`
	Entries []Entry `json:"entries"`
}

type Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Hash        string `json:"hash"`
	Salt        string `json:"salt"`
}

type Entry struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	URL      string    `json:"url"`
	Notes    string    `json:"notes"`
	Created  time.Time `json:"created"`
}

func ReadConfigFile() AppConfig {
	var config AppConfig

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	conf_filepath := filepath.Join(dirname, "/.config", "go_pwd_manager.toml")

	var k = koanf.New("/")
	if err := k.Load(file.Provider(conf_filepath), toml.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	k.Unmarshal("", &config)

	config.DBsFolder = strings.Replace(config.DBsFolder, "~", dirname, 1)

	return config
}

func ReadDBsFolder(folderPath string) ([]string, error) {
	var temp []string
	data, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	for _, s := range data {
		ext := s.Name()[strings.LastIndex(s.Name(), ".")+1:]
		if ext == "json" {
			temp = append(temp, s.Name())
		}
	}

	return temp, nil
}

func IsFileHashValid(filename, masterPassword string) (isOk bool, salt []byte, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false, nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Парсим JSON
	var passwordFile PasswordFile
	err = json.Unmarshal(data, &passwordFile)
	if err != nil {
		return false, nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	hash := MakeHash(passwordFile.Database.Meta.Name, masterPassword)
	if fmt.Sprintf("%x", hash) != passwordFile.Database.Meta.Hash {
		fmt.Print("not equal")
		return false, nil, nil
	} else {
		return true, []byte(passwordFile.Database.Meta.Salt), nil
	}
}

// Функция для чтения файла
func ReadPasswordFile(filename string, key []byte) ([]table.Row, error) {
	var decryptedData []table.Row

	//TODO filePath := filepath.Join(config.folder, filename)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Парсим JSON
	var passwordFile PasswordFile
	err = json.Unmarshal(data, &passwordFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	for _, d := range passwordFile.Database.Entries {
		dec_d, err := DecryptAES256(d.Password, key)
		if err != nil {
			// err = err
			break
		}
		decryptedData = append(decryptedData, table.Row{d.Title, dec_d})
	}

	return decryptedData, nil
}

func AddToPasswordFile(dbsFolder, filename, title, password string, key []byte) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Парсим JSON
	var passwordFile PasswordFile
	err = json.Unmarshal(data, &passwordFile)
	if err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	hashedPassword, err := EncryptAES256([]byte(password), key)
	if err != nil {
		return err
	}

	entry := Entry{ID: uuid.NewString(), Title: title, Password: hashedPassword}

	passwordFile.Database.Entries = append(passwordFile.Database.Entries, entry)

	newData, err := json.MarshalIndent(passwordFile, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %v", err)
	}

	err = os.WriteFile(filepath.Join(dbsFolder, filename), newData, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return err
	}
	return nil
}

func RemoveFromPasswordFile(dbsFolder, filename string, selectedIndex int) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Парсим JSON
	var passwordFile PasswordFile
	err = json.Unmarshal(data, &passwordFile)
	if err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Find and remove entry
	entries := passwordFile.Database.Entries
	passwordFile.Database.Entries = append(entries[:selectedIndex], entries[selectedIndex+1:]...)

	newData, err := json.MarshalIndent(passwordFile, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %v", err)
	}

	err = os.WriteFile(filepath.Join(dbsFolder, filename), newData, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return err
	}
	return nil
}

// hash its db title hashed with password
func CreatePasswordFile(filename string, dbsFolder string, masterPassword string) error {
	filename = filename + ".json"

	hash := MakeHash(filename, masterPassword)

	salt, err := GenerateSalt()
	if err != nil {
		return fmt.Errorf("ошибка генерации соли: %v", err)
	}

	// Хэшируем мастер-пароль

	db := &PasswordFile{
		Header: Header{
			Crypto: Crypto{
				Cipher:      "AES-256",
				Compression: "GZip",
			},
		},
		Database: Database{
			Meta: Meta{
				Name:        filename,
				Description: "Personal password database",
				Hash:        fmt.Sprintf("%x", hash),
				Salt:        fmt.Sprintf("%x", salt), // Соль в hex
			},
			Entries: []Entry{}, // Пустой массив entries
		},
	}

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %v", err)
	}

	err = os.WriteFile(filepath.Join(dbsFolder, filename), data, 0600)
	if err != nil {
		return fmt.Errorf("ошибка записи файла: %v", err)
	}

	return nil
}
