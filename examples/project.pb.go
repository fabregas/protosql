// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1-devel
// 	protoc        v3.19.4
// source: test.proto

package main

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ProjectStatus int32

const (
	ProjectStatus_PROJECT_STATUS_UNDEFINED ProjectStatus = 0
	ProjectStatus_PROJECT_STATUS_INITIAL   ProjectStatus = 1
	ProjectStatus_PROJECT_STATUS_ACTIVE    ProjectStatus = 2
	ProjectStatus_PROJECT_STATUS_BLOCKED   ProjectStatus = 3
)

// Enum value maps for ProjectStatus.
var (
	ProjectStatus_name = map[int32]string{
		0: "PROJECT_STATUS_UNDEFINED",
		1: "PROJECT_STATUS_INITIAL",
		2: "PROJECT_STATUS_ACTIVE",
		3: "PROJECT_STATUS_BLOCKED",
	}
	ProjectStatus_value = map[string]int32{
		"PROJECT_STATUS_UNDEFINED": 0,
		"PROJECT_STATUS_INITIAL":   1,
		"PROJECT_STATUS_ACTIVE":    2,
		"PROJECT_STATUS_BLOCKED":   3,
	}
)

func (x ProjectStatus) Enum() *ProjectStatus {
	p := new(ProjectStatus)
	*p = x
	return p
}


func (x ProjectStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}


type Project struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                      int32                  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Name                    string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Website                 string                 `protobuf:"bytes,3,opt,name=website,proto3" json:"website,omitempty"`
	Description             string                 `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	Status                  ProjectStatus          `protobuf:"varint,7,opt,name=status,proto3,enum=tnxwallet.personal.v1.ProjectStatus" json:"status,omitempty"`
	CreateTime              *timestamppb.Timestamp `protobuf:"bytes,10,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	UpdateTime              *timestamppb.Timestamp `protobuf:"bytes,11,opt,name=update_time,json=updateTime,proto3" json:"update_time,omitempty"`
	MerchantName            string                 `protobuf:"bytes,12,opt,name=merchant_name,json=merchantName,proto3" json:"merchant_name,omitempty"`
	MerchantCountry         string                 `protobuf:"bytes,13,opt,name=merchant_country,json=merchantCountry,proto3" json:"merchant_country,omitempty"`
	RefundPolicyLink        string                 `protobuf:"bytes,14,opt,name=refund_policy_link,json=refundPolicyLink,proto3" json:"refund_policy_link,omitempty"`
	CustomerServiceFeedback string                 `protobuf:"bytes,15,opt,name=customer_service_feedback,json=customerServiceFeedback,proto3" json:"customer_service_feedback,omitempty"`
}


func (*Project) Reset() {}
func (*Project) ProtoMessage() {}

func (x *Project) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Project) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Project) GetWebsite() string {
	if x != nil {
		return x.Website
	}
	return ""
}

func (x *Project) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Project) GetStatus() ProjectStatus {
	if x != nil {
		return x.Status
	}
	return ProjectStatus_PROJECT_STATUS_UNDEFINED
}


func (x *Project) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

func (x *Project) GetUpdateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.UpdateTime
	}
	return nil
}

func (x *Project) GetMerchantName() string {
	if x != nil {
		return x.MerchantName
	}
	return ""
}

func (x *Project) GetMerchantCountry() string {
	if x != nil {
		return x.MerchantCountry
	}
	return ""
}

func (x *Project) GetRefundPolicyLink() string {
	if x != nil {
		return x.RefundPolicyLink
	}
	return ""
}

func (x *Project) GetCustomerServiceFeedback() string {
	if x != nil {
		return x.CustomerServiceFeedback
	}
	return ""
}