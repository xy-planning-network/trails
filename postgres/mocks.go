// Code generated by MockGen. DO NOT EDIT.
// Source: service.go

// Package postgres is a generated GoMock package.
package postgres

import (
	gomock "github.com/golang/mock/gomock"
	gorm "gorm.io/gorm"
	reflect "reflect"
)

// MockDatabaseService is a mock of DatabaseService interface
type MockDatabaseService struct {
	ctrl     *gomock.Controller
	recorder *MockDatabaseServiceMockRecorder
}

// MockDatabaseServiceMockRecorder is the mock recorder for MockDatabaseService
type MockDatabaseServiceMockRecorder struct {
	mock *MockDatabaseService
}

// NewMockDatabaseService creates a new mock instance
func NewMockDatabaseService(ctrl *gomock.Controller) *MockDatabaseService {
	mock := &MockDatabaseService{ctrl: ctrl}
	mock.recorder = &MockDatabaseServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDatabaseService) EXPECT() *MockDatabaseServiceMockRecorder {
	return m.recorder
}

// CountByQuery mocks base method
func (m *MockDatabaseService) CountByQuery(model interface{}, query map[string]interface{}) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountByQuery", model, query)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CountByQuery indicates an expected call of CountByQuery
func (mr *MockDatabaseServiceMockRecorder) CountByQuery(model, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountByQuery", reflect.TypeOf((*MockDatabaseService)(nil).CountByQuery), model, query)
}

// FetchByQuery mocks base method
func (m *MockDatabaseService) FetchByQuery(models interface{}, query string, params []interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchByQuery", models, query, params)
	ret0, _ := ret[0].(error)
	return ret0
}

// FetchByQuery indicates an expected call of FetchByQuery
func (mr *MockDatabaseServiceMockRecorder) FetchByQuery(models, query, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchByQuery", reflect.TypeOf((*MockDatabaseService)(nil).FetchByQuery), models, query, params)
}

// FindByID mocks base method
func (m *MockDatabaseService) FindByID(model, ID interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByID", model, ID)
	ret0, _ := ret[0].(error)
	return ret0
}

// FindByID indicates an expected call of FindByID
func (mr *MockDatabaseServiceMockRecorder) FindByID(model, ID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByID", reflect.TypeOf((*MockDatabaseService)(nil).FindByID), model, ID)
}

// FindByQuery mocks base method
func (m *MockDatabaseService) FindByQuery(model interface{}, query map[string]interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByQuery", model, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// FindByQuery indicates an expected call of FindByQuery
func (mr *MockDatabaseServiceMockRecorder) FindByQuery(model, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByQuery", reflect.TypeOf((*MockDatabaseService)(nil).FindByQuery), model, query)
}

// PagedByQuery mocks base method
func (m *MockDatabaseService) PagedByQuery(models interface{}, query string, params []interface{}, order string, page, perPage int, preloads ...string) (PagedData, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{models, query, params, order, page, perPage}
	for _, a := range preloads {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PagedByQuery", varargs...)
	ret0, _ := ret[0].(PagedData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PagedByQuery indicates an expected call of PagedByQuery
func (mr *MockDatabaseServiceMockRecorder) PagedByQuery(models, query, params, order, page, perPage interface{}, preloads ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{models, query, params, order, page, perPage}, preloads...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PagedByQuery", reflect.TypeOf((*MockDatabaseService)(nil).PagedByQuery), varargs...)
}

// PagedByQueryFromSession mocks base method
func (m *MockDatabaseService) PagedByQueryFromSession(models interface{}, session *gorm.DB, page, perPage int) (PagedData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PagedByQueryFromSession", models, session, page, perPage)
	ret0, _ := ret[0].(PagedData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PagedByQueryFromSession indicates an expected call of PagedByQueryFromSession
func (mr *MockDatabaseServiceMockRecorder) PagedByQueryFromSession(models, session, page, perPage interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PagedByQueryFromSession", reflect.TypeOf((*MockDatabaseService)(nil).PagedByQueryFromSession), models, session, page, perPage)
}
