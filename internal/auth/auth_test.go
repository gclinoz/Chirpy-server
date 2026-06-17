package auth

import "testing"

func TestHashPassword(t *testing.T) {
	pswd := "onceUponAtime"

	hash, err := HashPassword(pswd)
	if err != nil {
		t.Errorf("Error when creating hashes: %v", err)
	}

	match, err := CheckPasswordHash(pswd, hash)
	if err != nil {
		t.Errorf("Something wrong during comparsion: %v", err)
	}
	if match != true {
		t.Errorf("Hashed string does not match")
	}

	match, err = CheckPasswordHash("wrongpassword", hash)
	if err != nil {
		t.Errorf("Something wrong during comparsion: %v", err)
	}
	if match == true {
		t.Errorf("Hashed string match on wrong password")
	}
}
