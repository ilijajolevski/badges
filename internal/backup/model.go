package backup

import (
	"database/sql"
	"time"

	"github.com/finki/badges/internal/database"
)

// BackupDocument is the top-level structure of a backup JSON file.
type BackupDocument struct {
	Metadata BackupMetadata `json:"metadata"`
	Data     BackupData     `json:"data"`
}

// BackupMetadata contains information about when/how the backup was created.
type BackupMetadata struct {
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	CreatedBy   string `json:"created_by"`
	Application string `json:"application"`
}

// BackupData holds all database table data.
type BackupData struct {
	Roles   []RoleDTO   `json:"roles"`
	Users   []UserDTO   `json:"users"`
	APIKeys []APIKeyDTO `json:"api_keys"`
	Badges  []BadgeDTO  `json:"badges"`
}

// RoleDTO is the JSON-serializable representation of a database.Role.
type RoleDTO struct {
	RoleID      string `json:"role_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Permissions string `json:"permissions"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// UserDTO is the JSON-serializable representation of a database.User.
type UserDTO struct {
	UserID         string  `json:"user_id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	PasswordHash   string  `json:"password_hash"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	RoleID         string  `json:"role_id"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	LastLogin      *string `json:"last_login"`
	Status         string  `json:"status"`
	FailedAttempts int     `json:"failed_attempts"`
}

// APIKeyDTO is the JSON-serializable representation of a database.APIKey.
type APIKeyDTO struct {
	APIKeyID       string  `json:"api_key_id"`
	UserID         string  `json:"user_id"`
	APIKey         string  `json:"api_key"`
	Name           string  `json:"name"`
	Permissions    string  `json:"permissions"`
	CreatedAt      string  `json:"created_at"`
	ExpiresAt      string  `json:"expires_at"`
	LastUsed       *string `json:"last_used"`
	Status         string  `json:"status"`
	IPRestrictions string  `json:"ip_restrictions"`
}

// BadgeDTO is the JSON-serializable representation of a database.Badge.
// Binary image fields (JPG/PNG) are excluded — they can be regenerated.
type BadgeDTO struct {
	CommitID        string  `json:"commit_id"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	Issuer          string  `json:"issuer"`
	IssueDate       string  `json:"issue_date"`
	SoftwareName    string  `json:"software_name"`
	SoftwareVersion string  `json:"software_version"`
	SoftwareURL     *string `json:"software_url"`
	Notes           *string `json:"notes"`
	SVGContent      *string `json:"svg_content"`
	ExpiryDate      *string `json:"expiry_date"`
	IssuerURL       *string `json:"issuer_url"`
	CustomConfig    *string `json:"custom_config"`
	LastReview      *string `json:"last_review"`
	CoveredVersion  *string `json:"covered_version"`
	RepositoryLink  *string `json:"repository_link"`
	PublicNote      *string `json:"public_note"`
	InternalNote    *string `json:"internal_note"`
	ContactDetails  *string `json:"contact_details"`
	CertificateName *string `json:"certificate_name"`
	SpecialtyDomain *string `json:"specialty_domain"`
	SoftwareSCID    *string `json:"software_sc_id"`
	SoftwareSCURL   *string `json:"software_sc_url"`
}

const timeFormat = time.RFC3339

// --- Role conversion ---

func rolesToDTOs(roles []*database.Role) []RoleDTO {
	dtos := make([]RoleDTO, len(roles))
	for i, r := range roles {
		dtos[i] = RoleDTO{
			RoleID:      r.RoleID,
			Name:        r.Name,
			Description: r.Description,
			Permissions: r.Permissions,
			CreatedAt:   r.CreatedAt.Format(timeFormat),
			UpdatedAt:   r.UpdatedAt.Format(timeFormat),
		}
	}
	return dtos
}

func dtosToRoles(dtos []RoleDTO) ([]*database.Role, error) {
	roles := make([]*database.Role, len(dtos))
	for i, d := range dtos {
		createdAt, err := time.Parse(timeFormat, d.CreatedAt)
		if err != nil {
			return nil, err
		}
		updatedAt, err := time.Parse(timeFormat, d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		roles[i] = &database.Role{
			RoleID:      d.RoleID,
			Name:        d.Name,
			Description: d.Description,
			Permissions: d.Permissions,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}
	}
	return roles, nil
}

// --- User conversion ---

func usersToDTOs(users []*database.User) []UserDTO {
	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dto := UserDTO{
			UserID:         u.UserID,
			Username:       u.Username,
			Email:          u.Email,
			PasswordHash:   u.PasswordHash,
			FirstName:      u.FirstName,
			LastName:       u.LastName,
			RoleID:         u.RoleID,
			CreatedAt:      u.CreatedAt.Format(timeFormat),
			UpdatedAt:      u.UpdatedAt.Format(timeFormat),
			Status:         u.Status,
			FailedAttempts: u.FailedAttempts,
		}
		if u.LastLogin.Valid {
			s := u.LastLogin.Time.Format(timeFormat)
			dto.LastLogin = &s
		}
		dtos[i] = dto
	}
	return dtos
}

func dtosToUsers(dtos []UserDTO) ([]*database.User, error) {
	users := make([]*database.User, len(dtos))
	for i, d := range dtos {
		createdAt, err := time.Parse(timeFormat, d.CreatedAt)
		if err != nil {
			return nil, err
		}
		updatedAt, err := time.Parse(timeFormat, d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		u := &database.User{
			UserID:         d.UserID,
			Username:       d.Username,
			Email:          d.Email,
			PasswordHash:   d.PasswordHash,
			FirstName:      d.FirstName,
			LastName:       d.LastName,
			RoleID:         d.RoleID,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			Status:         d.Status,
			FailedAttempts: d.FailedAttempts,
		}
		if d.LastLogin != nil {
			t, err := time.Parse(timeFormat, *d.LastLogin)
			if err != nil {
				return nil, err
			}
			u.LastLogin = sql.NullTime{Time: t, Valid: true}
		}
		users[i] = u
	}
	return users, nil
}

// --- APIKey conversion ---

func apiKeysToDTOs(keys []*database.APIKey) []APIKeyDTO {
	dtos := make([]APIKeyDTO, len(keys))
	for i, k := range keys {
		dto := APIKeyDTO{
			APIKeyID:       k.APIKeyID,
			UserID:         k.UserID,
			APIKey:         k.APIKey,
			Name:           k.Name,
			Permissions:    k.Permissions,
			CreatedAt:      k.CreatedAt.Format(timeFormat),
			ExpiresAt:      k.ExpiresAt.Format(timeFormat),
			Status:         k.Status,
			IPRestrictions: k.IPRestrictions,
		}
		if k.LastUsed.Valid {
			s := k.LastUsed.Time.Format(timeFormat)
			dto.LastUsed = &s
		}
		dtos[i] = dto
	}
	return dtos
}

func dtosToAPIKeys(dtos []APIKeyDTO) ([]*database.APIKey, error) {
	keys := make([]*database.APIKey, len(dtos))
	for i, d := range dtos {
		createdAt, err := time.Parse(timeFormat, d.CreatedAt)
		if err != nil {
			return nil, err
		}
		expiresAt, err := time.Parse(timeFormat, d.ExpiresAt)
		if err != nil {
			return nil, err
		}
		k := &database.APIKey{
			APIKeyID:       d.APIKeyID,
			UserID:         d.UserID,
			APIKey:         d.APIKey,
			Name:           d.Name,
			Permissions:    d.Permissions,
			CreatedAt:      createdAt,
			ExpiresAt:      expiresAt,
			Status:         d.Status,
			IPRestrictions: d.IPRestrictions,
		}
		if d.LastUsed != nil {
			t, err := time.Parse(timeFormat, *d.LastUsed)
			if err != nil {
				return nil, err
			}
			k.LastUsed = sql.NullTime{Time: t, Valid: true}
		}
		keys[i] = k
	}
	return keys, nil
}

// --- Badge conversion ---

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func ptrToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func badgesToDTOs(badges []*database.Badge) []BadgeDTO {
	dtos := make([]BadgeDTO, len(badges))
	for i, b := range badges {
		dtos[i] = BadgeDTO{
			CommitID:        b.CommitID,
			Type:            b.Type,
			Status:          b.Status,
			Issuer:          b.Issuer,
			IssueDate:       b.IssueDate,
			SoftwareName:    b.SoftwareName,
			SoftwareVersion: b.SoftwareVersion,
			SoftwareURL:     nullStringToPtr(b.SoftwareURL),
			Notes:           nullStringToPtr(b.Notes),
			SVGContent:      nullStringToPtr(b.SVGContent),
			ExpiryDate:      nullStringToPtr(b.ExpiryDate),
			IssuerURL:       nullStringToPtr(b.IssuerURL),
			CustomConfig:    nullStringToPtr(b.CustomConfig),
			LastReview:      nullStringToPtr(b.LastReview),
			CoveredVersion:  nullStringToPtr(b.CoveredVersion),
			RepositoryLink:  nullStringToPtr(b.RepositoryLink),
			PublicNote:      nullStringToPtr(b.PublicNote),
			InternalNote:    nullStringToPtr(b.InternalNote),
			ContactDetails:  nullStringToPtr(b.ContactDetails),
			CertificateName: nullStringToPtr(b.CertificateName),
			SpecialtyDomain: nullStringToPtr(b.SpecialtyDomain),
			SoftwareSCID:    nullStringToPtr(b.SoftwareSCID),
			SoftwareSCURL:   nullStringToPtr(b.SoftwareSCURL),
		}
	}
	return dtos
}

func dtosToBadges(dtos []BadgeDTO) []*database.Badge {
	badges := make([]*database.Badge, len(dtos))
	for i, d := range dtos {
		badges[i] = &database.Badge{
			CommitID:        d.CommitID,
			Type:            d.Type,
			Status:          d.Status,
			Issuer:          d.Issuer,
			IssueDate:       d.IssueDate,
			SoftwareName:    d.SoftwareName,
			SoftwareVersion: d.SoftwareVersion,
			SoftwareURL:     ptrToNullString(d.SoftwareURL),
			Notes:           ptrToNullString(d.Notes),
			SVGContent:      ptrToNullString(d.SVGContent),
			ExpiryDate:      ptrToNullString(d.ExpiryDate),
			IssuerURL:       ptrToNullString(d.IssuerURL),
			CustomConfig:    ptrToNullString(d.CustomConfig),
			LastReview:      ptrToNullString(d.LastReview),
			CoveredVersion:  ptrToNullString(d.CoveredVersion),
			RepositoryLink:  ptrToNullString(d.RepositoryLink),
			PublicNote:      ptrToNullString(d.PublicNote),
			InternalNote:    ptrToNullString(d.InternalNote),
			ContactDetails:  ptrToNullString(d.ContactDetails),
			CertificateName: ptrToNullString(d.CertificateName),
			SpecialtyDomain: ptrToNullString(d.SpecialtyDomain),
			SoftwareSCID:    ptrToNullString(d.SoftwareSCID),
			SoftwareSCURL:   ptrToNullString(d.SoftwareSCURL),
		}
	}
	return badges
}
