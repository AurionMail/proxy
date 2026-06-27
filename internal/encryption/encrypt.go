package encryption

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/openpgp"
)

// Encrypt chiffre le message avec la clé publique PGP (format ASCII-armored ou binaire).
// v1: implémentation simple, sans gestion avancée des préférences.
func Encrypt(pubKeys map[string]string, raw []byte) ([]byte, error) {
	var combinedKeys openpgp.EntityList

	for _, armoredKey := range pubKeys {
		entityList, err := readPubKey(armoredKey)
		if err != nil {
			return nil, fmt.Errorf("pgp: invalid public key: %w", err)
		}
		// On ajoute les entités de chaque clé à la liste globale
		combinedKeys = append(combinedKeys, entityList...)
	}

	if len(combinedKeys) == 0 {
		return raw, nil // Rien à chiffrer
	}

	var buf bytes.Buffer
	// openpgp.Encrypt va chiffrer pour TOUTES les entités présentes dans combinedKeys
	w, err := openpgp.Encrypt(&buf, combinedKeys, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("pgp: encrypt: %w", err)
	}

	if _, err := w.Write(raw); err != nil {
		return nil, fmt.Errorf("pgp: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("pgp: close: %w", err)
	}

	return buf.Bytes(), nil
}

func readPubKey(armored string) (openpgp.EntityList, error) {
	r := bytes.NewBufferString(armored)

	// On tente d'abord ASCII-armored, puis binaire.
	el, err := openpgp.ReadArmoredKeyRing(r)
	if err == nil {
		return el, nil
	}

	r2 := bytes.NewBufferString(armored)
	el, err = openpgp.ReadKeyRing(r2)
	if err != nil {
		return nil, err
	}
	return el, nil
}
