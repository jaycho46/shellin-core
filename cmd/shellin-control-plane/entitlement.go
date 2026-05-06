// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"net/http"

	"github.com/jaycho46/shellin-core/controlplane"
)

func (s *controlPlaneServer) validateEntitlement(r *http.Request) (controlplane.UserEntitlement, error) {
	claims, err := validateAccessRequest(r, s.deps.accessValidator)
	if err != nil {
		return controlplane.UserEntitlement{}, err
	}
	entitlement, ok := s.deps.store.LookupBySubject(claims.Subject)
	if !ok || !isEntitlementUsable(entitlement) {
		return controlplane.UserEntitlement{}, errors.New("unauthorized")
	}
	return entitlement, nil
}
