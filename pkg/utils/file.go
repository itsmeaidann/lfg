package utils

import (
	"encoding/json"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// load the exchange's symbol map to map universal symbols to local symbols
func LoadExchangeSymbolMap(exchange string) map[string]string {
	bytes, err := os.ReadFile(filepath.Join("pkg", "exchange", exchange, "config", "symbol.json"))
	if err != nil {
		log.Fatalf("fail to read symbol map file: %v", err)
	}
	var symbolMap map[string]string
	if err := json.Unmarshal(bytes, &symbolMap); err != nil {
		log.Fatalf("fail to unmarshal symbol map: %v", err)
	}
	return symbolMap
}
