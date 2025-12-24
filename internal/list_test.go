package internal

import (
	"context"
	"io"
	"strings"
	"testing"
)

// MockS3Client for testing list functionality
type mockS3ClientForList struct {
	files map[string][]byte
}

func newMockS3ClientForList() *mockS3ClientForList {
	return &mockS3ClientForList{
		files: make(map[string][]byte),
	}
}

func (m *mockS3ClientForList) Upload(ctx context.Context, bucket, key string, data io.Reader) error {
	content, _ := io.ReadAll(data)
	m.files[key] = content
	return nil
}

func (m *mockS3ClientForList) UploadWithProgress(ctx context.Context, bucket, key string, data io.Reader, size int64, description string) error {
	return m.Upload(ctx, bucket, key, data)
}

func (m *mockS3ClientForList) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, exists := m.files[key]
	return exists, nil
}

func (m *mockS3ClientForList) Download(ctx context.Context, bucket, key string) ([]byte, error) {
	return m.files[key], nil
}

func (m *mockS3ClientForList) DownloadStream(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	data := m.files[key]
	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (m *mockS3ClientForList) Copy(ctx context.Context, bucket, srcKey, dstKey string) error {
	m.files[dstKey] = m.files[srcKey]
	return nil
}

func (m *mockS3ClientForList) Delete(ctx context.Context, bucket, key string) error {
	delete(m.files, key)
	return nil
}

func (m *mockS3ClientForList) List(ctx context.Context, bucket, prefix string) ([]string, error) {
	var keys []string
	for key := range m.files {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func TestListImages(t *testing.T) {
	ctx := context.Background()
	mock := newMockS3ClientForList()

	// Add some test images
	mock.files["images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz"] = []byte("image1")
	mock.files["images/myapp/202507/myapp-20250721-2118-f7a5a27.json"] = []byte("{}")
	mock.files["images/myapp/202507/myapp-20250720-1045-abc1234.tar.gz"] = []byte("image2")
	mock.files["images/myapp/202506/myapp-20250615-0930-def5678.tar.gz"] = []byte("image3")
	mock.files["images/otherapp/202507/otherapp-20250721-1200-xyz9999.tar.gz"] = []byte("image4")

	listService := NewListService(mock, "test-bucket")

	// Test listing all images for myapp
	images, err := listService.ListImages(ctx, "myapp", "")
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}

	if len(images) != 3 {
		t.Errorf("Expected 3 images, got %d", len(images))
	}

	// Test listing images for specific month
	images, err = listService.ListImages(ctx, "myapp", "202507")
	if err != nil {
		t.Fatalf("ListImages with month filter failed: %v", err)
	}

	if len(images) != 2 {
		t.Errorf("Expected 2 images for 202507, got %d", len(images))
	}
}

func TestListTags(t *testing.T) {
	ctx := context.Background()
	mock := newMockS3ClientForList()

	// Add some test tags
	tagData := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz",
		"source_image": "myapp:20250721-2118-f7a5a27"
	}`
	mock.files["tags/myapp/v1.0.0.json"] = []byte(tagData)
	mock.files["tags/myapp/v1.1.0.json"] = []byte(tagData)
	mock.files["tags/myapp/v2.0.0.json"] = []byte(tagData)

	listService := NewListService(mock, "test-bucket")

	tags, err := listService.ListTags(ctx, "myapp")
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}

	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	// Check that tags are sorted (descending by version string)
	if tags[0].Version != "v2.0.0" {
		t.Errorf("Expected first tag to be v2.0.0, got %s", tags[0].Version)
	}
}

func TestListEnvironments(t *testing.T) {
	ctx := context.Background()
	mock := newMockS3ClientForList()

	// Add environment pointers
	prodPointer := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz",
		"source_image": "myapp:20250721-2118-f7a5a27"
	}`
	stagingPointer := `{
		"target_type": "tag",
		"target_path": "tags/myapp/v1.0.0.json",
		"source_image": "myapp:20250720-1045-abc1234",
		"source_tag": "v1.0.0"
	}`
	mock.files["pointers/myapp/production.json"] = []byte(prodPointer)
	mock.files["pointers/myapp/staging.json"] = []byte(stagingPointer)

	listService := NewListService(mock, "test-bucket")

	envs, err := listService.ListEnvironments(ctx, "myapp")
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(envs))
	}

	// Check production
	var prodEnv, stagingEnv *EnvInfo
	for i := range envs {
		if envs[i].Environment == "production" {
			prodEnv = &envs[i]
		}
		if envs[i].Environment == "staging" {
			stagingEnv = &envs[i]
		}
	}

	if prodEnv == nil {
		t.Fatal("Production environment not found")
	}
	if prodEnv.TargetType != TargetTypeImage {
		t.Errorf("Expected production target type 'image', got '%s'", prodEnv.TargetType)
	}

	if stagingEnv == nil {
		t.Fatal("Staging environment not found")
	}
	if stagingEnv.TargetType != TargetTypeTag {
		t.Errorf("Expected staging target type 'tag', got '%s'", stagingEnv.TargetType)
	}
	if stagingEnv.SourceTag != "v1.0.0" {
		t.Errorf("Expected staging source tag 'v1.0.0', got '%s'", stagingEnv.SourceTag)
	}
}

func TestListApps(t *testing.T) {
	ctx := context.Background()
	mock := newMockS3ClientForList()

	// Add files for multiple apps
	mock.files["images/app1/202507/app1-20250721-2118-f7a5a27.tar.gz"] = []byte("image")
	mock.files["tags/app2/v1.0.0.json"] = []byte("{}")
	mock.files["pointers/app3/production.json"] = []byte("{}")

	listService := NewListService(mock, "test-bucket")

	apps, err := listService.ListApps(ctx)
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}

	if len(apps) != 3 {
		t.Errorf("Expected 3 apps, got %d", len(apps))
	}

	// Check that apps are sorted
	expected := []string{"app1", "app2", "app3"}
	for i, app := range apps {
		if app != expected[i] {
			t.Errorf("Expected app[%d] to be %s, got %s", i, expected[i], app)
		}
	}
}

func TestGetTagForEnvironment(t *testing.T) {
	ctx := context.Background()
	mock := newMockS3ClientForList()

	// Add environment pointer promoted via tag
	stagingPointer := `{
		"target_type": "tag",
		"target_path": "tags/myapp/v1.0.0.json",
		"source_image": "myapp:20250720-1045-abc1234",
		"source_tag": "v1.0.0"
	}`
	mock.files["pointers/myapp/staging.json"] = []byte(stagingPointer)

	// Add environment pointer promoted directly
	prodPointer := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz",
		"source_image": "myapp:20250721-2118-f7a5a27"
	}`
	mock.files["pointers/myapp/production.json"] = []byte(prodPointer)

	listService := NewListService(mock, "test-bucket")

	// Test getting tag for staging (promoted via tag)
	tag, err := listService.GetTagForEnvironment(ctx, "myapp", "staging")
	if err != nil {
		t.Fatalf("GetTagForEnvironment failed: %v", err)
	}
	if tag != "v1.0.0" {
		t.Errorf("Expected tag 'v1.0.0', got '%s'", tag)
	}

	// Test getting tag for production (promoted directly - should return empty)
	tag, err = listService.GetTagForEnvironment(ctx, "myapp", "production")
	if err != nil {
		t.Fatalf("GetTagForEnvironment failed: %v", err)
	}
	if tag != "" {
		t.Errorf("Expected empty tag for direct promotion, got '%s'", tag)
	}
}
