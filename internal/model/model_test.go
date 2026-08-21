package model

import "testing"

func TestSignupActiveAndCategory(t *testing.T) {
	if !SignupApproved.IsActive() {
		t.Fatal("approved active")
	}
	if SignupRejected.IsActive() {
		t.Fatal("rejected not active")
	}
	if !ValidCategory(CatEnvironment) {
		t.Fatal("env")
	}
	if ValidCategory("nope") {
		t.Fatal("invalid")
	}
	if CategoryName(CatElderly) != "助老" {
		t.Fatal(CategoryName(CatElderly))
	}
}

func TestUserWriteGuard(t *testing.T) {
	u := User{Role: RoleVolunteer, Status: UserFrozen}
	if err := u.CanWrite(); err != ErrAccountFrozen {
		t.Fatalf("%v", err)
	}
	u.Status = UserBanned
	if err := u.CanWrite(); err != ErrAccountBanned {
		t.Fatalf("%v", err)
	}
}
