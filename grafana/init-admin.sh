#!/bin/bash

# Warte bis Grafana lädt
sleep 10

# Setze Admin-Passwort
grafana-cli admin reset-admin-password admin

echo "Admin-Passwort gesetzt: admin/admin"
