// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"net/http"

	"github.com/GenesisKernel/go-genesis/packages/consts"
	"github.com/GenesisKernel/go-genesis/packages/converter"
	"github.com/GenesisKernel/go-genesis/packages/model"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func (m Mode) getEcosystemParamHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)
	m.EcosysIDValidator.SetLogger(logger)
	form := &ecosystemForm{
		Validator: m.EcosysIDValidator,
	}

	if err := parseForm(r, form); err != nil {
		errorResponse(w, err, http.StatusBadRequest)
		return
	}

	params := mux.Vars(r)

	sp := &model.StateParameter{}
	sp.SetTablePrefix(form.EcosystemPrefix)
	name := params["name"]

	if found, err := sp.Get(nil, name); err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("Getting state parameter by name")
		errorResponse(w, err)
		return
	} else if !found {
		logger.WithFields(log.Fields{"type": consts.NotFound, "key": name}).Error("state parameter not found")
		errorResponse(w, errParamNotFound.Errorf(name))
		return
	}

	jsonResponse(w, &paramResult{
		ID:         converter.Int64ToStr(sp.ID),
		Name:       sp.Name,
		Value:      sp.Value,
		Conditions: sp.Conditions,
	})
}

func getEcosystemNameHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	ecosystemID := converter.StrToInt64(r.FormValue("id"))
	ecosystems := model.Ecosystem{}
	found, err := ecosystems.Get(ecosystemID)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("on getting ecosystem name")
		errorResponse(w, err)
		return
	}
	if !found {
		logger.WithFields(log.Fields{"type": consts.NotFound, "ecosystem_id": ecosystemID}).Error("ecosystem by id not found")
		errorResponse(w, errParamNotFound.Errorf("name"))
		return
	}

	jsonResponse(w, &struct {
		EcosystemName string `json:"ecosystem_name"`
	}{
		EcosystemName: ecosystems.Name,
	})
}
