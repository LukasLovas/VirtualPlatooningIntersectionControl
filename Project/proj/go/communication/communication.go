package network

import (
	"encoding/json"
	"fmt"
	"net"
)

func ReceiveVehicleData(conn net.Conn) (map[string]map[string]interface{}, error) {
	lenBuf := make([]byte, 4)
	_, err := conn.Read(lenBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	msgLen := (int(lenBuf[0]) << 24) | (int(lenBuf[1]) << 16) | (int(lenBuf[2]) << 8) | int(lenBuf[3])

	buf := make([]byte, msgLen)
	bytesRead := 0
	for bytesRead < msgLen {
		n, err := conn.Read(buf[bytesRead:])
		if err != nil {
			return nil, fmt.Errorf("failed to read message: %w", err)
		}
		bytesRead += n
	}

	var vehicleData map[string]map[string]interface{}
	err = json.Unmarshal(buf, &vehicleData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return vehicleData, nil
}

func SendCommands(conn net.Conn, commands map[string]interface{}) error {
	data, err := json.Marshal(commands)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	msgLen := len(data)
	lenBuf := []byte{
		byte(msgLen >> 24),
		byte(msgLen >> 16),
		byte(msgLen >> 8),
		byte(msgLen),
	}

	_, err = conn.Write(lenBuf)
	if err != nil {
		return fmt.Errorf("failed to send message length: %w", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
