package filearea

import (
	"database/sql"
	"fmt"
)

// Repo handles database operations for file areas and entries.
type Repo struct {
	db *sql.DB
}

// NewRepo creates a new file area repository.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// ListAreas returns all file areas the user has access to download from.
func (r *Repo) ListAreas(userLevel int) ([]*Area, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.name, a.description, a.disk_path, a.download_level, a.upload_level, a.sort_order,
		       COALESCE((SELECT COUNT(*) FROM file_entries WHERE area_id = a.id), 0) as file_count
		FROM file_areas a
		WHERE a.download_level <= ?
		ORDER BY a.sort_order, a.name
	`, userLevel)
	if err != nil {
		return nil, fmt.Errorf("list file areas: %w", err)
	}
	defer rows.Close()

	var areas []*Area
	for rows.Next() {
		a := &Area{}
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.DiskPath,
			&a.DownloadLevel, &a.UploadLevel, &a.SortOrder, &a.FileCount); err != nil {
			return nil, err
		}
		areas = append(areas, a)
	}
	return areas, rows.Err()
}

// GetArea returns a single area by ID.
func (r *Repo) GetArea(id int) (*Area, error) {
	a := &Area{}
	err := r.db.QueryRow(`
		SELECT id, name, description, disk_path, download_level, upload_level, sort_order
		FROM file_areas WHERE id = ?
	`, id).Scan(&a.ID, &a.Name, &a.Description, &a.DiskPath,
		&a.DownloadLevel, &a.UploadLevel, &a.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("get area %d: %w", id, err)
	}
	return a, nil
}

// ListFiles returns files in an area, paginated.
func (r *Repo) ListFiles(areaID, offset, limit int) ([]*Entry, error) {
	rows, err := r.db.Query(`
		SELECT f.id, f.area_id, f.filename, f.description, f.size_bytes,
		       f.uploader_id, COALESCE(u.username, 'Unknown') as uploader_name,
		       f.download_count, f.uploaded_at
		FROM file_entries f
		LEFT JOIN users u ON u.id = f.uploader_id
		WHERE f.area_id = ?
		ORDER BY f.filename
		LIMIT ? OFFSET ?
	`, areaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		e := &Entry{}
		if err := rows.Scan(&e.ID, &e.AreaID, &e.Filename, &e.Description,
			&e.SizeBytes, &e.UploaderID, &e.UploaderName,
			&e.DownloadCount, &e.UploadedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetFile returns a single file entry by ID.
func (r *Repo) GetFile(id int) (*Entry, error) {
	e := &Entry{}
	err := r.db.QueryRow(`
		SELECT f.id, f.area_id, f.filename, f.description, f.size_bytes,
		       f.uploader_id, COALESCE(u.username, 'Unknown') as uploader_name,
		       f.download_count, f.uploaded_at
		FROM file_entries f
		LEFT JOIN users u ON u.id = f.uploader_id
		WHERE f.id = ?
	`, id).Scan(&e.ID, &e.AreaID, &e.Filename, &e.Description,
		&e.SizeBytes, &e.UploaderID, &e.UploaderName,
		&e.DownloadCount, &e.UploadedAt)
	if err != nil {
		return nil, fmt.Errorf("get file %d: %w", id, err)
	}
	return e, nil
}

// FindByName searches for files by name pattern across all areas.
func (r *Repo) FindByName(pattern string, userLevel int) ([]*Entry, error) {
	rows, err := r.db.Query(`
		SELECT f.id, f.area_id, f.filename, f.description, f.size_bytes,
		       f.uploader_id, COALESCE(u.username, 'Unknown') as uploader_name,
		       f.download_count, f.uploaded_at
		FROM file_entries f
		LEFT JOIN users u ON u.id = f.uploader_id
		JOIN file_areas a ON a.id = f.area_id
		WHERE (f.filename LIKE ? OR f.description LIKE ?)
		  AND a.download_level <= ?
		ORDER BY f.filename
		LIMIT 50
	`, "%"+pattern+"%", "%"+pattern+"%", userLevel)
	if err != nil {
		return nil, fmt.Errorf("search files: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		e := &Entry{}
		if err := rows.Scan(&e.ID, &e.AreaID, &e.Filename, &e.Description,
			&e.SizeBytes, &e.UploaderID, &e.UploaderName,
			&e.DownloadCount, &e.UploadedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// AddEntry creates a new file entry record.
func (r *Repo) AddEntry(areaID int, filename, description string, sizeBytes int64, uploaderID int) (int, error) {
	result, err := r.db.Exec(`
		INSERT INTO file_entries (area_id, filename, description, size_bytes, uploader_id)
		VALUES (?, ?, ?, ?, ?)
	`, areaID, filename, description, sizeBytes, uploaderID)
	if err != nil {
		return 0, fmt.Errorf("add file entry: %w", err)
	}
	id, err := result.LastInsertId()
	return int(id), err
}

// IncrementDownload increments the download count for a file.
func (r *Repo) IncrementDownload(fileID int) error {
	_, err := r.db.Exec(`
		UPDATE file_entries SET download_count = download_count + 1 WHERE id = ?
	`, fileID)
	return err
}
