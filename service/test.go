package service

import "goweb_staging/model"

func (s *Service) Test() (data model.Test, err error) {
	data, err = s.dao.Test()
	return
}
