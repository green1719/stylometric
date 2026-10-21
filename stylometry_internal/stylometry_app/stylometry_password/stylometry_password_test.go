package password

import "testing"

func TestHashAndCheck(t *testing.T) {
	hash, err := Hash("TestPass-2026!")
	if err != nil {
		t.Fatalf("не удалось создать хеш: %v", err)
	}
	if hash == "TestPass-2026!" {
		t.Fatal("вместо хеша сохранён исходный пароль")
	}
	if !Check("TestPass-2026!", hash) {
		t.Fatal("правильный пароль не прошёл проверку")
	}
	if Check("wrong-password", hash) {
		t.Fatal("неправильный пароль прошёл проверку")
	}
}
