package utils

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var DB *sql.DB

type MessageObj struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

func InitDB(dbPath string) {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(`
        		CREATE TABLE IF NOT EXISTS messages (
        			conversation_id TEXT,
        			message_index INTEGER,
        			role TEXT,
        			name TEXT,
        			content TEXT,
        			PRIMARY KEY (conversation_id, message_index)
        		)
        	`)
	if err != nil {
		log.Fatal(err)
	}
}

func SaveMessage(conversationID string, messageIndex int, role, name,
	content string) error {
	_, err := DB.Exec(
		"INSERT INTO messages (conversation_id, message_index, role, name, content) VALUES (?, ?, ?, ?, ?)",
		conversationID,
		messageIndex,
		role,
		name,
		content,
	)
	return err
}
func GetNextMessageIndex(conversationID string) (int, error) {
	var maxIndex int
	err := DB.QueryRow("SELECT COALESCE(MAX(message_index), -1) FROM messages WHERE conversation_id = ?", conversationID).Scan(&maxIndex)
	if err != nil {
		return 0, err
	}
	return maxIndex + 1, nil
}

func GetConversationHistory(conversationID string) ([]MessageObj, error) {
	rows, err := DB.Query(`
                SELECT role, name, content
                FROM messages
                WHERE conversation_id = ?
                ORDER BY message_index DESC
                LIMIT 6
            `, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageObj
	for rows.Next() {
		var msg MessageObj
		err := rows.Scan(&msg.Role, &msg.Name, &msg.Content)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Reverse the order of messages
	for i := 0; i < len(messages)/2; i++ {
		j := len(messages) - 1 - i
		messages[i], messages[j] = messages[j], messages[i]
	}

	var result []MessageObj
	if len(messages) > 0 {
		// Always include the most recent message
		result = append(result, messages[len(messages)-1])

		// Then ensure alternating messages for the rest
		userTurn := messages[len(messages)-1].Role != "user"
		for i := len(messages) - 2; i >= 0 && len(result) < 6; i-- {
			if (userTurn && messages[i].Role == "user") || (!userTurn && messages[i].Role == "assistant") {
				result = append([]MessageObj{messages[i]}, result...)
				userTurn = !userTurn
			}
		}
	}

	return result, nil
}
