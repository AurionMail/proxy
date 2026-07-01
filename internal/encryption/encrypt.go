package encryption

import (
	"bytes"
	"fmt"
	"log"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

func Encrypt(pubKeys map[string]string, raw []byte) ([]byte, error) {
	log.Printf("[DEBUG] Encrypting message body...")
	var combinedKeys openpgp.EntityList

	for _, armoredKey := range pubKeys {
		entityList, err := readPubKey(armoredKey)
		if err != nil {
			return nil, fmt.Errorf("pgp: invalid public key: %w", err)
		}
		combinedKeys = append(combinedKeys, entityList...)
	}

	if len(combinedKeys) == 0 {
		return raw, nil
	}

	var outBuf bytes.Buffer

	// Encodage au format ASCII-Armor (-----BEGIN PGP MESSAGE-----)
	armorWriter, err := armor.Encode(&outBuf, "PGP MESSAGE", nil)
	if err != nil {
		return nil, fmt.Errorf("pgp: armor init: %w", err)
	}

	// Initialisation du chiffrement OpenPGP connecté à l'armure ASCII
	w, err := openpgp.Encrypt(armorWriter, combinedKeys, nil, nil, nil)
	if err != nil {
		armorWriter.Close()
		return nil, fmt.Errorf("pgp: encrypt: %w", err)
	}

	if _, err := w.Write(raw); err != nil {
		w.Close()
		armorWriter.Close()
		return nil, fmt.Errorf("pgp: write: %w", err)
	}

	w.Close()
	armorWriter.Close()

	return outBuf.Bytes(), nil
}

func readPubKey(armored string) (openpgp.EntityList, error) {
	log.Printf("[DEBUG] Reading public key: %s", armored)
	r := bytes.NewBufferString(armored)

	// On tente d'abord ASCII-armored, puis binaire.
	el, err := openpgp.ReadArmoredKeyRing(r)
	if err == nil {
		return el, nil
	} else {
		log.Printf("[DEBUG] Reading public key ERROR: %s", err)
	}

	r2 := bytes.NewBufferString(armored)
	el, err = openpgp.ReadKeyRing(r2)
	if err != nil {
		return nil, err
	} else {
		log.Printf("[DEBUG] Reading public key ERROR 2: %s", err)
	}
	return el, nil
}
