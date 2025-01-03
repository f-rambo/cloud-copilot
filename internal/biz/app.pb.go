// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.27.1
// source: internal/biz/app.proto

package biz

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	"gorm.io/gorm"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type AppReleaseSatus int32

const (
	AppReleaseSatus_APP_RELEASE_PENDING AppReleaseSatus = 0
	AppReleaseSatus_APP_RELEASE_RUNNING AppReleaseSatus = 1
	AppReleaseSatus_APP_RELEASE_FAILED  AppReleaseSatus = 2
)

// Enum value maps for AppReleaseSatus.
var (
	AppReleaseSatus_name = map[int32]string{
		0: "APP_RELEASE_PENDING",
		1: "APP_RELEASE_RUNNING",
		2: "APP_RELEASE_FAILED",
	}
	AppReleaseSatus_value = map[string]int32{
		"APP_RELEASE_PENDING": 0,
		"APP_RELEASE_RUNNING": 1,
		"APP_RELEASE_FAILED":  2,
	}
)

func (x AppReleaseSatus) Enum() *AppReleaseSatus {
	p := new(AppReleaseSatus)
	*p = x
	return p
}

func (x AppReleaseSatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AppReleaseSatus) Descriptor() protoreflect.EnumDescriptor {
	return file_internal_biz_app_proto_enumTypes[0].Descriptor()
}

func (AppReleaseSatus) Type() protoreflect.EnumType {
	return &file_internal_biz_app_proto_enumTypes[0]
}

func (x AppReleaseSatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AppReleaseSatus.Descriptor instead.
func (AppReleaseSatus) EnumDescriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{0}
}

type AppType struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// @goimport: "gorm.io/gorm"
	// @gofield: gorm.Model
	Id          int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`                       // @gotags: gorm:"column:id;primaryKey;AUTO_INCREMENT"
	Name        string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`                      // @gotags: gorm:"column:name; default:''; NOT NULL"
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"` // @gotags: gorm:"column:description; default:''; NOT NULL"
	gorm.Model
}

func (x *AppType) Reset() {
	*x = AppType{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_biz_app_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppType) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppType) ProtoMessage() {}

func (x *AppType) ProtoReflect() protoreflect.Message {
	mi := &file_internal_biz_app_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppType.ProtoReflect.Descriptor instead.
func (*AppType) Descriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{0}
}

func (x *AppType) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *AppType) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *AppType) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

type AppRepo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// @goimport: "gorm.io/gorm"
	// @gofield: gorm.Model
	Id          int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`                                   // @gotags: gorm:"column:id;primaryKey;AUTO_INCREMENT"
	Name        string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`                                  // @gotags: gorm:"column:name; default:''; NOT NULL"
	Url         string `protobuf:"bytes,3,opt,name=url,proto3" json:"url,omitempty" gorm:"column:url; default:''; NOT NULL"`                                     // @gotags: gorm:"column:url; default:''; NOT NULL"
	IndexPath   string `protobuf:"bytes,4,opt,name=index_path,json=indexPath,proto3" json:"index_path,omitempty" gorm:"column:index_path; default:''; NOT NULL"` // @gotags: gorm:"column:index_path; default:''; NOT NULL"
	Description string `gorm:"column:description; default:''; NOT NULL" protobuf:"bytes,5,opt,name=description,proto3" json:"description,omitempty"`             // @gotags: gorm:"column:description; default:''; NOT NULL"
	gorm.Model
}

func (x *AppRepo) Reset() {
	*x = AppRepo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_biz_app_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppRepo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppRepo) ProtoMessage() {}

func (x *AppRepo) ProtoReflect() protoreflect.Message {
	mi := &file_internal_biz_app_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppRepo.ProtoReflect.Descriptor instead.
func (*AppRepo) Descriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{1}
}

func (x *AppRepo) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *AppRepo) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *AppRepo) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *AppRepo) GetIndexPath() string {
	if x != nil {
		return x.IndexPath
	}
	return ""
}

func (x *AppRepo) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

type AppVersion struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// @goimport: "gorm.io/gorm"
	// @gofield: gorm.Model
	Id            int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`                                                   // @gotags: gorm:"column:id;primaryKey;AUTO_INCREMENT"
	AppId         int64  `json:"app_id,omitempty" gorm:"column:app_id; default:0; NOT NULL; index" protobuf:"varint,2,opt,name=app_id,json=appId,proto3"`                          // @gotags: gorm:"column:app_id; default:0; NOT NULL; index"
	Name          string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`                                                  // @gotags: gorm:"column:name; default:''; NOT NULL"
	Chart         string `protobuf:"bytes,4,opt,name=chart,proto3" json:"chart,omitempty" gorm:"column:chart; default:''; NOT NULL"`                                               // @gotags: gorm:"column:chart; default:''; NOT NULL" // as file path
	Version       string `protobuf:"bytes,5,opt,name=version,proto3" json:"version,omitempty" gorm:"column:version; default:''; NOT NULL; index"`                                  // @gotags: gorm:"column:version; default:''; NOT NULL; index"
	DefaultConfig string `protobuf:"bytes,6,opt,name=default_config,json=defaultConfig,proto3" json:"default_config,omitempty" gorm:"column:default_config; default:''; NOT NULL"` // @gotags: gorm:"column:default_config; default:''; NOT NULL"
	gorm.Model
}

func (x *AppVersion) Reset() {
	*x = AppVersion{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_biz_app_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppVersion) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppVersion) ProtoMessage() {}

func (x *AppVersion) ProtoReflect() protoreflect.Message {
	mi := &file_internal_biz_app_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppVersion.ProtoReflect.Descriptor instead.
func (*AppVersion) Descriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{2}
}

func (x *AppVersion) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *AppVersion) GetAppId() int64 {
	if x != nil {
		return x.AppId
	}
	return 0
}

func (x *AppVersion) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *AppVersion) GetChart() string {
	if x != nil {
		return x.Chart
	}
	return ""
}

func (x *AppVersion) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *AppVersion) GetDefaultConfig() string {
	if x != nil {
		return x.DefaultConfig
	}
	return ""
}

type App struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// @goimport: "gorm.io/gorm"
	// @gofield: gorm.Model
	Id          int64         `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`                                      // @gotags: gorm:"column:id;primaryKey;AUTO_INCREMENT"
	Name        string        `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty" gorm:"column:name; default:''; NOT NULL; index"`                              // @gotags: gorm:"column:name; default:''; NOT NULL; index"
	Icon        string        `protobuf:"bytes,3,opt,name=icon,proto3" json:"icon,omitempty" gorm:"column:icon; default:''; NOT NULL"`                                     // @gotags: gorm:"column:icon; default:''; NOT NULL"
	AppTypeId   int64         `protobuf:"varint,4,opt,name=app_type_id,json=appTypeId,proto3" json:"app_type_id,omitempty" gorm:"column:app_type_id; default:0; NOT NULL"` // @gotags: gorm:"column:app_type_id; default:0; NOT NULL"
	AppRepoId   int64         `gorm:"column:app_repo_id; default:0; NOT NULL" protobuf:"varint,5,opt,name=app_repo_id,json=appRepoId,proto3" json:"app_repo_id,omitempty"` // @gotags: gorm:"column:app_repo_id; default:0; NOT NULL"
	Description string        `protobuf:"bytes,6,opt,name=description,proto3" json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`                // @gotags: gorm:"column:description; default:''; NOT NULL"
	Versions    []*AppVersion `protobuf:"bytes,7,rep,name=versions,proto3" json:"versions,omitempty" gorm:"-"`                                                             // @gotags: gorm:"-"
	Readme      string        `protobuf:"bytes,8,opt,name=readme,proto3" json:"readme,omitempty" gorm:"-"`                                                                 // @gotags: gorm:"-"
	Metadata    []byte        `json:"metadata,omitempty" gorm:"-" protobuf:"bytes,9,opt,name=metadata,proto3"`
	gorm.Model                // @gotags: gorm:"-"
}

func (x *App) Reset() {
	*x = App{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_biz_app_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *App) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*App) ProtoMessage() {}

func (x *App) ProtoReflect() protoreflect.Message {
	mi := &file_internal_biz_app_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use App.ProtoReflect.Descriptor instead.
func (*App) Descriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{3}
}

func (x *App) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *App) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *App) GetIcon() string {
	if x != nil {
		return x.Icon
	}
	return ""
}

func (x *App) GetAppTypeId() int64 {
	if x != nil {
		return x.AppTypeId
	}
	return 0
}

func (x *App) GetAppRepoId() int64 {
	if x != nil {
		return x.AppRepoId
	}
	return 0
}

func (x *App) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *App) GetVersions() []*AppVersion {
	if x != nil {
		return x.Versions
	}
	return nil
}

func (x *App) GetReadme() string {
	if x != nil {
		return x.Readme
	}
	return ""
}

func (x *App) GetMetadata() []byte {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type AppReleaseResource struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// @goimport: "gorm.io/gorm"
	// @gofield: gorm.Model
	Id         int64    `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`                                          // @gotags: gorm:"column:id;primaryKey;AUTO_INCREMENT"
	ReleaseId  int64    `protobuf:"varint,2,opt,name=release_id,json=releaseId,proto3" json:"release_id,omitempty" gorm:"column:release_id; default:0; NOT NULL; index"` // @gotags: gorm:"column:release_id; default:0; NOT NULL; index"
	Name       string   `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`                                         // @gotags: gorm:"column:name; default:''; NOT NULL"
	Kind       string   `protobuf:"bytes,4,opt,name=kind,proto3" json:"kind,omitempty" gorm:"column:kind; default:''; NOT NULL"`                                         // @gotags: gorm:"column:kind; default:''; NOT NULL"
	Manifest   string   `protobuf:"bytes,5,opt,name=manifest,proto3" json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"`                             // @gotags: gorm:"column:manifest; default:''; NOT NULL"
	StartedAt  string   `protobuf:"bytes,6,opt,name=started_at,json=startedAt,proto3" json:"started_at,omitempty" gorm:"column:started_at; default:''; NOT NULL"`        // @gotags: gorm:"column:started_at; default:''; NOT NULL"
	Events     []string `protobuf:"bytes,7,rep,name=events,proto3" json:"events,omitempty" gorm:"-"`                                                                     // @gotags: gorm:"-"
	Status     []string `protobuf:"bytes,8,rep,name=status,proto3" json:"status,omitempty" gorm:"-"`
	gorm.Model          // @gotags: gorm:"-"
}

func (x *AppReleaseResource) Reset() {
	*x = AppReleaseResource{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_biz_app_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppReleaseResource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppReleaseResource) ProtoMessage() {}

func (x *AppReleaseResource) ProtoReflect() protoreflect.Message {
	mi := &file_internal_biz_app_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppReleaseResource.ProtoReflect.Descriptor instead.
func (*AppReleaseResource) Descriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{4}
}

func (x *AppReleaseResource) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *AppReleaseResource) GetReleaseId() int64 {
	if x != nil {
		return x.ReleaseId
	}
	return 0
}

func (x *AppReleaseResource) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *AppReleaseResource) GetKind() string {
	if x != nil {
		return x.Kind
	}
	return ""
}

func (x *AppReleaseResource) GetManifest() string {
	if x != nil {
		return x.Manifest
	}
	return ""
}

func (x *AppReleaseResource) GetStartedAt() string {
	if x != nil {
		return x.StartedAt
	}
	return ""
}

func (x *AppReleaseResource) GetEvents() []string {
	if x != nil {
		return x.Events
	}
	return nil
}

func (x *AppReleaseResource) GetStatus() []string {
	if x != nil {
		return x.Status
	}
	return nil
}

type AppRelease struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// @goimport: "gorm.io/gorm"
	// @gofield: gorm.Model
	Id          int64                 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`                                           // @gotags: gorm:"column:id;primaryKey;AUTO_INCREMENT"
	ReleaseName string                `protobuf:"bytes,2,opt,name=release_name,json=releaseName,proto3" json:"release_name,omitempty" gorm:"column:release_name; default:''; NOT NULL"` // @gotags: gorm:"column:release_name; default:''; NOT NULL"
	AppId       int64                 `protobuf:"varint,3,opt,name=app_id,json=appId,proto3" json:"app_id,omitempty" gorm:"column:app_id; default:0; NOT NULL; index"`                  // @gotags: gorm:"column:app_id; default:0; NOT NULL; index"
	VersionId   int64                 `protobuf:"varint,4,opt,name=version_id,json=versionId,proto3" json:"version_id,omitempty" gorm:"column:version_id; default:0; NOT NULL; index"`  // @gotags: gorm:"column:version_id; default:0; NOT NULL; index"
	ClusterId   int64                 `protobuf:"varint,5,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" gorm:"column:cluster_id; default:0; NOT NULL; index"`  // @gotags: gorm:"column:cluster_id; default:0; NOT NULL; index"
	ProjectId   int64                 `protobuf:"varint,6,opt,name=project_id,json=projectId,proto3" json:"project_id,omitempty" gorm:"column:project_id; default:0; NOT NULL; index"`  // @gotags: gorm:"column:project_id; default:0; NOT NULL; index"
	UserId      int64                 `protobuf:"varint,7,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty" gorm:"column:user_id; default:0; NOT NULL; index"`              // @gotags: gorm:"column:user_id; default:0; NOT NULL; index"
	Namespace   string                `protobuf:"bytes,8,opt,name=namespace,proto3" json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`                           // @gotags: gorm:"column:namespace; default:''; NOT NULL"
	Config      string                `protobuf:"bytes,9,opt,name=config,proto3" json:"config,omitempty" gorm:"column:config; default:''; NOT NULL"`                                    // @gotags: gorm:"column:config; default:''; NOT NULL"
	Status      AppReleaseSatus       `protobuf:"varint,10,opt,name=status,proto3,enum=biz.app.AppReleaseSatus" json:"status,omitempty" gorm:"column:status; default:0; NOT NULL"`      // @gotags: gorm:"column:status; default:0; NOT NULL"
	Manifest    string                `protobuf:"bytes,11,opt,name=manifest,proto3" json:"manifest,omitempty" gorm:"column:manifest; default:''; NOT NULL"`                             // @gotags: gorm:"column:manifest; default:''; NOT NULL"
	Notes       string                `gorm:"column:notes; default:''; NOT NULL" protobuf:"bytes,12,opt,name=notes,proto3" json:"notes,omitempty"`                                      // @gotags: gorm:"column:notes; default:''; NOT NULL"
	Logs        string                `protobuf:"bytes,13,opt,name=logs,proto3" json:"logs,omitempty" gorm:"column:logs; default:''; NOT NULL"`                                         // @gotags: gorm:"column:logs; default:''; NOT NULL"
	Dryrun      bool                  `json:"dryrun,omitempty" gorm:"column:dryrun; default:false; NOT NULL" protobuf:"varint,14,opt,name=dryrun,proto3"`                               // @gotags: gorm:"column:dryrun; default:false; NOT NULL"
	Atomic      bool                  `protobuf:"varint,15,opt,name=atomic,proto3" json:"atomic,omitempty" gorm:"column:atomic; default:false; NOT NULL"`                               // @gotags: gorm:"column:atomic; default:false; NOT NULL"
	Wait        bool                  `protobuf:"varint,16,opt,name=wait,proto3" json:"wait,omitempty" gorm:"column:wait; default:false; NOT NULL"`                                     // @gotags: gorm:"column:wait; default:false; NOT NULL"
	Resources   []*AppReleaseResource `protobuf:"bytes,17,rep,name=resources,proto3" json:"resources,omitempty" gorm:"-"`
	gorm.Model                        // @gotags: gorm:"-"
}

func (x *AppRelease) Reset() {
	*x = AppRelease{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_biz_app_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppRelease) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppRelease) ProtoMessage() {}

func (x *AppRelease) ProtoReflect() protoreflect.Message {
	mi := &file_internal_biz_app_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppRelease.ProtoReflect.Descriptor instead.
func (*AppRelease) Descriptor() ([]byte, []int) {
	return file_internal_biz_app_proto_rawDescGZIP(), []int{5}
}

func (x *AppRelease) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *AppRelease) GetReleaseName() string {
	if x != nil {
		return x.ReleaseName
	}
	return ""
}

func (x *AppRelease) GetAppId() int64 {
	if x != nil {
		return x.AppId
	}
	return 0
}

func (x *AppRelease) GetVersionId() int64 {
	if x != nil {
		return x.VersionId
	}
	return 0
}

func (x *AppRelease) GetClusterId() int64 {
	if x != nil {
		return x.ClusterId
	}
	return 0
}

func (x *AppRelease) GetProjectId() int64 {
	if x != nil {
		return x.ProjectId
	}
	return 0
}

func (x *AppRelease) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *AppRelease) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *AppRelease) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}

func (x *AppRelease) GetStatus() AppReleaseSatus {
	if x != nil {
		return x.Status
	}
	return AppReleaseSatus_APP_RELEASE_PENDING
}

func (x *AppRelease) GetManifest() string {
	if x != nil {
		return x.Manifest
	}
	return ""
}

func (x *AppRelease) GetNotes() string {
	if x != nil {
		return x.Notes
	}
	return ""
}

func (x *AppRelease) GetLogs() string {
	if x != nil {
		return x.Logs
	}
	return ""
}

func (x *AppRelease) GetDryrun() bool {
	if x != nil {
		return x.Dryrun
	}
	return false
}

func (x *AppRelease) GetAtomic() bool {
	if x != nil {
		return x.Atomic
	}
	return false
}

func (x *AppRelease) GetWait() bool {
	if x != nil {
		return x.Wait
	}
	return false
}

func (x *AppRelease) GetResources() []*AppReleaseResource {
	if x != nil {
		return x.Resources
	}
	return nil
}

var File_internal_biz_app_proto protoreflect.FileDescriptor

var file_internal_biz_app_proto_rawDesc = []byte{
	0x0a, 0x16, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x62, 0x69, 0x7a, 0x2f, 0x61,
	0x70, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x62, 0x69, 0x7a, 0x2e, 0x61, 0x70,
	0x70, 0x22, 0x4f, 0x0a, 0x07, 0x41, 0x70, 0x70, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x22, 0x80, 0x01, 0x0a, 0x07, 0x41, 0x70, 0x70, 0x52, 0x65, 0x70, 0x6f, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x75, 0x72, 0x6c, 0x12, 0x1d, 0x0a, 0x0a, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x5f, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x50,
	0x61, 0x74, 0x68, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x9e, 0x01, 0x0a, 0x0a, 0x41, 0x70, 0x70, 0x56, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x15, 0x0a, 0x06, 0x61, 0x70, 0x70, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x61, 0x70, 0x70, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x14, 0x0a, 0x05, 0x63, 0x68, 0x61, 0x72, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x63, 0x68, 0x61, 0x72, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12,
	0x25, 0x0a, 0x0e, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0x84, 0x02, 0x0a, 0x03, 0x41, 0x70, 0x70, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x69, 0x63, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x69, 0x63, 0x6f, 0x6e, 0x12, 0x1e, 0x0a, 0x0b, 0x61, 0x70, 0x70, 0x5f, 0x74, 0x79,
	0x70, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x61, 0x70, 0x70,
	0x54, 0x79, 0x70, 0x65, 0x49, 0x64, 0x12, 0x1e, 0x0a, 0x0b, 0x61, 0x70, 0x70, 0x5f, 0x72, 0x65,
	0x70, 0x6f, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x61, 0x70, 0x70,
	0x52, 0x65, 0x70, 0x6f, 0x49, 0x64, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2f, 0x0a, 0x08, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62, 0x69, 0x7a,
	0x2e, 0x61, 0x70, 0x70, 0x2e, 0x41, 0x70, 0x70, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x52,
	0x08, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x61,
	0x64, 0x6d, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x61, 0x64, 0x6d,
	0x65, 0x12, 0x1a, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x09, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x22, 0xd6, 0x01,
	0x0a, 0x12, 0x41, 0x70, 0x70, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x52, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f,
	0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73,
	0x65, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6b, 0x69, 0x6e, 0x64, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6b, 0x69, 0x6e, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x6d,
	0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6d,
	0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x74, 0x61, 0x72, 0x74,
	0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x74, 0x61,
	0x72, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73,
	0x18, 0x07, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x16,
	0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06,
	0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0xf9, 0x03, 0x0a, 0x0a, 0x41, 0x70, 0x70, 0x52, 0x65,
	0x6c, 0x65, 0x61, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x02, 0x69, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x72, 0x65, 0x6c,
	0x65, 0x61, 0x73, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x15, 0x0a, 0x06, 0x61, 0x70, 0x70, 0x5f,
	0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x61, 0x70, 0x70, 0x49, 0x64, 0x12,
	0x1d, 0x0a, 0x0a, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x09, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x1d,
	0x0a, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1d, 0x0a,
	0x0a, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07,
	0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75,
	0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x09, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x30, 0x0a, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x18, 0x2e, 0x62, 0x69,
	0x7a, 0x2e, 0x61, 0x70, 0x70, 0x2e, 0x41, 0x70, 0x70, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65,
	0x53, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1a, 0x0a,
	0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x74,
	0x65, 0x73, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x6f, 0x74, 0x65, 0x73, 0x12,
	0x12, 0x0a, 0x04, 0x6c, 0x6f, 0x67, 0x73, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6c,
	0x6f, 0x67, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x72, 0x79, 0x72, 0x75, 0x6e, 0x18, 0x0e, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x06, 0x64, 0x72, 0x79, 0x72, 0x75, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x61,
	0x74, 0x6f, 0x6d, 0x69, 0x63, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x61, 0x74, 0x6f,
	0x6d, 0x69, 0x63, 0x12, 0x12, 0x0a, 0x04, 0x77, 0x61, 0x69, 0x74, 0x18, 0x10, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x04, 0x77, 0x61, 0x69, 0x74, 0x12, 0x39, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x73, 0x18, 0x11, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x62, 0x69, 0x7a,
	0x2e, 0x61, 0x70, 0x70, 0x2e, 0x41, 0x70, 0x70, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x52,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x73, 0x2a, 0x5b, 0x0a, 0x0f, 0x41, 0x70, 0x70, 0x52, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65,
	0x53, 0x61, 0x74, 0x75, 0x73, 0x12, 0x17, 0x0a, 0x13, 0x41, 0x50, 0x50, 0x5f, 0x52, 0x45, 0x4c,
	0x45, 0x41, 0x53, 0x45, 0x5f, 0x50, 0x45, 0x4e, 0x44, 0x49, 0x4e, 0x47, 0x10, 0x00, 0x12, 0x17,
	0x0a, 0x13, 0x41, 0x50, 0x50, 0x5f, 0x52, 0x45, 0x4c, 0x45, 0x41, 0x53, 0x45, 0x5f, 0x52, 0x55,
	0x4e, 0x4e, 0x49, 0x4e, 0x47, 0x10, 0x01, 0x12, 0x16, 0x0a, 0x12, 0x41, 0x50, 0x50, 0x5f, 0x52,
	0x45, 0x4c, 0x45, 0x41, 0x53, 0x45, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44, 0x10, 0x02, 0x42,
	0x30, 0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x66, 0x2d,
	0x72, 0x61, 0x6d, 0x62, 0x6f, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2d, 0x63, 0x6f, 0x70, 0x69,
	0x6c, 0x6f, 0x74, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x62, 0x69, 0x7a,
	0x3b, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_biz_app_proto_rawDescOnce sync.Once
	file_internal_biz_app_proto_rawDescData = file_internal_biz_app_proto_rawDesc
)

func file_internal_biz_app_proto_rawDescGZIP() []byte {
	file_internal_biz_app_proto_rawDescOnce.Do(func() {
		file_internal_biz_app_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_biz_app_proto_rawDescData)
	})
	return file_internal_biz_app_proto_rawDescData
}

var file_internal_biz_app_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_internal_biz_app_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_internal_biz_app_proto_goTypes = []any{
	(AppReleaseSatus)(0),       // 0: biz.app.AppReleaseSatus
	(*AppType)(nil),            // 1: biz.app.AppType
	(*AppRepo)(nil),            // 2: biz.app.AppRepo
	(*AppVersion)(nil),         // 3: biz.app.AppVersion
	(*App)(nil),                // 4: biz.app.App
	(*AppReleaseResource)(nil), // 5: biz.app.AppReleaseResource
	(*AppRelease)(nil),         // 6: biz.app.AppRelease
}
var file_internal_biz_app_proto_depIdxs = []int32{
	3, // 0: biz.app.App.versions:type_name -> biz.app.AppVersion
	0, // 1: biz.app.AppRelease.status:type_name -> biz.app.AppReleaseSatus
	5, // 2: biz.app.AppRelease.resources:type_name -> biz.app.AppReleaseResource
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_internal_biz_app_proto_init() }
func file_internal_biz_app_proto_init() {
	if File_internal_biz_app_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_biz_app_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*AppType); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_biz_app_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*AppRepo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_biz_app_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*AppVersion); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_biz_app_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*App); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_biz_app_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*AppReleaseResource); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_biz_app_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*AppRelease); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_biz_app_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_biz_app_proto_goTypes,
		DependencyIndexes: file_internal_biz_app_proto_depIdxs,
		EnumInfos:         file_internal_biz_app_proto_enumTypes,
		MessageInfos:      file_internal_biz_app_proto_msgTypes,
	}.Build()
	File_internal_biz_app_proto = out.File
	file_internal_biz_app_proto_rawDesc = nil
	file_internal_biz_app_proto_goTypes = nil
	file_internal_biz_app_proto_depIdxs = nil
}
