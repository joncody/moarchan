use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use base64::Engine;
use rand::RngCore;
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use crate::error::AppError;

pub struct SessionStore {
    cipher: Aes256Gcm,
    pub cookie_name: String,
    pub max_age_secs: u32,
    pub secure: bool,
}

impl SessionStore {
    pub fn new(cookie_name: &str, hash_key: &str, block_key: &str, secure: bool) -> Self {
        let mut hasher = Sha256::new();
        hasher.update(format!("{block_key}{hash_key}").as_bytes());
        let derived_key = hasher.finalize();

        let cipher = Aes256Gcm::new_from_slice(&derived_key)
            .expect("Derived key is guaranteed 32 bytes for AES-256");

        Self {
            cipher,
            cookie_name: cookie_name.to_string(),
            max_age_secs: 86400,
            secure,
        }
    }

    pub fn cookie_value(&self, token: &str) -> String {
        let secure_flag = if self.secure { "; Secure" } else { "" };
        format!(
            "{}={token}; Path=/; Max-Age={}; HttpOnly; SameSite=Lax{secure_flag}",
            self.cookie_name, self.max_age_secs
        )
    }

    pub fn delete_cookie_value(&self) -> String {
        let secure_flag = if self.secure { "; Secure" } else { "" };
        format!(
            "{}=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax{secure_flag}",
            self.cookie_name
        )
    }

    pub fn encrypt(&self, values: &HashMap<String, String>) -> Result<String, AppError> {
        let payload = serde_json::to_vec(values)
            .map_err(|e| AppError::Internal(anyhow::anyhow!("Serialize session: {e}")))?;

        let mut nonce_bytes = [0u8; 12];
        rand::thread_rng().fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        let ciphertext = self.cipher.encrypt(nonce, payload.as_ref())
            .map_err(|e| AppError::Internal(anyhow::anyhow!("AES-GCM encrypt error: {e}")))?;

        let mut combined = Vec::with_capacity(12 + ciphertext.len());
        combined.extend_from_slice(&nonce_bytes);
        combined.extend_from_slice(&ciphertext);

        Ok(base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(combined))
    }

    pub fn decrypt(&self, encoded: &str) -> Result<HashMap<String, String>, AppError> {
        let raw = base64::engine::general_purpose::URL_SAFE_NO_PAD
            .decode(encoded)
            .map_err(|_| AppError::Unauthorized("Malformed session token".into()))?;

        if raw.len() < 12 {
            return Err(AppError::Unauthorized("Session token too short".into()));
        }

        let (nonce_bytes, ciphertext) = raw.split_at(12);
        let nonce = Nonce::from_slice(nonce_bytes);

        let decrypted = self.cipher.decrypt(nonce, ciphertext)
            .map_err(|_| AppError::Unauthorized("Invalid or tampered session token".into()))?;

        let values = serde_json::from_slice(&decrypted)
            .map_err(|_| AppError::Unauthorized("Corrupted session JSON payload".into()))?;

        Ok(values)
    }
}
