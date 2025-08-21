package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
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
	Key         string `json:"Key"`
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

func ReadDBsFolder() {}

// Функция для чтения файла
func ReadPasswordFile(filename string) (*PasswordFile, error) {
	// Читаем файл

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

	return &passwordFile, nil
}

func AddToPasswordFile(data any) error {
	return nil
}

func CreatePasswordFile(filename string, dbsFolder string, masterPassword string) error {
	filename = filename + ".json"

	salt, err := GenerateSalt()
	if err != nil {
		return fmt.Errorf("ошибка генерации соли: %v", err)
	}

	// Хэшируем мастер-пароль
	hash := GenerateKey(masterPassword, salt)

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
				Key:         fmt.Sprintf("%x", hash), // Хэш в hex
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
