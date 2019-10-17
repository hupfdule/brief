package address

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"poiu.de/brief/parser"
)

// Address represents a single address with several fields like
// fromName, fromAddress, bank, etc.
type Address struct {
	Fields map[string]parser.BrfLines
}

// ReadFromFile reads an address file and returns Address objects for the
// entries in the file.
//
// The returned map contains the name (or id) of the Adress as the key and
// the actual Address object as the value.
func ReadFromFile(filepath string) (map[string]Address, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("Error opening file %s for reading: %w", filepath, err)
	}
	defer file.Close()

	senderAddresses := make(map[string]Address)

	var curAddress *Address = nil

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		} else if strings.HasPrefix(line, "address:") {
			addressId := strings.TrimSpace(strings.TrimPrefix(line, "address:"))
			curAddress = new(Address)
			curAddress.Fields = make(map[string]parser.BrfLines)
			senderAddresses[addressId] = *curAddress
		} else {
			if curAddress == nil {
				return nil, fmt.Errorf("Invalid address file %s. content without 'address:': %s", filepath, scanner.Text())
			} else {
				s := strings.SplitN(line, ":", 2)
				if len(s) != 2 {
					return nil, fmt.Errorf("Invalid line in address file %s: %s", filepath, scanner.Text())
				}
				k := strings.TrimSpace(s[0])
				v := strings.TrimSpace(s[1])
				if curAddress.Fields[k] == nil {
					curAddress.Fields[k] = make([]string, 0)
				}
				curAddress.Fields[k] = append(curAddress.Fields[k], v)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error reading file %s: %w", filepath, err)
	}

	return senderAddresses, nil
}
