package organizer

import "testing"

func TestDefaultTemplates(t *testing.T) {
	if MovieFolderTemplate == "" || MovieFileTemplate == "" {
		t.Fatal("expected movie templates")
	}
	if TVFolderTemplate == "" || TVFileTemplate == "" {
		t.Fatal("expected tv templates")
	}
	if AVFolderTemplate == "" || AVFileTemplate == "" {
		t.Fatal("expected av templates")
	}
}
