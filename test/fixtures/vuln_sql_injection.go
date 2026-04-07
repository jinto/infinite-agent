// EVAL-VULN: sql-injection
// This file intentionally contains SQL injection vulnerabilities for eval testing.
// DO NOT use this code in production.

package vulnerable

import (
	"database/sql"
	"fmt"
	"net/http"
)

func GetUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	// VULN: Direct string concatenation in SQL query
	query := fmt.Sprintf("SELECT * FROM users WHERE username = '%s'", username)
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "DB error", 500)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, email string
		rows.Scan(&id, &name, &email)
		fmt.Fprintf(w, "User: %s (%s)\n", name, email)
	}
}

func DeleteUser(db *sql.DB, userID string) error {
	// VULN: Unsanitized user input in DELETE query
	_, err := db.Exec("DELETE FROM users WHERE id = " + userID)
	return err
}

func SearchUsers(db *sql.DB, searchTerm string) (*sql.Rows, error) {
	// VULN: LIKE clause injection
	query := "SELECT * FROM users WHERE name LIKE '%" + searchTerm + "%'"
	return db.Query(query)
}
