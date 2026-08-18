use std::sync::LazyLock;
use base64::Engine;
use regex::Regex;
use sha2::{Digest, Sha256};

static TAG_REGEX: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"&gt;&gt;([A-Za-z0-9]+)").unwrap());

pub fn sanitize_comment(raw: &str) -> (String, Vec<String>) {
    let escaped = html_escape::encode_safe(raw).replace("\r\n", "\n");
    let mut tags = Vec::new();
    let mut formatted_lines = Vec::new();

    for line in escaped.lines() {
        // Transform >>hash into tag spans
        let formatted_line = TAG_REGEX.replace_all(line, |caps: &regex::Captures| {
            let post_hash = &caps[1];
            tags.push(post_hash.to_string());
            format!(
                r#"<span class="post-tag blue-text-link" data-tag="{post_hash}">&gt;&gt;{post_hash}</span>"#
            )
        });

        // Greentext quotation (>quote)
        let final_line = if formatted_line.starts_with("&gt;") && !formatted_line.starts_with("&gt;&gt;") {
            format!(r#"<span class="post-quote">{formatted_line}</span>"#)
        } else {
            formatted_line.to_string()
        };

        formatted_lines.push(final_line);
    }

    (formatted_lines.join("<br />"), tags)
}

pub fn sanitize_name(raw: &str, secret_salt: &str) -> String {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return "Anonymous".to_string();
    }

    if let Some((name_part, secure_key)) = trimmed.split_once("##") {
        let display_name = if name_part.trim().is_empty() {
            "Anonymous"
        } else {
            name_part.trim()
        };
        let escaped_name = html_escape::encode_safe(display_name);

        let mut hasher = Sha256::new();
        hasher.update(format!("{secure_key}{secret_salt}").as_bytes());
        let raw_hash = hasher.finalize();
        let encoded = base64::engine::general_purpose::STANDARD_NO_PAD.encode(raw_hash);
        let trip = &encoded[..12.min(encoded.len())];

        format!(r#"{escaped_name} <span class="post-tripcode">!!{trip}</span>"#)
    } else if let Some((name_part, key)) = trimmed.split_once('#') {
        let display_name = if name_part.trim().is_empty() {
            "Anonymous"
        } else {
            name_part.trim()
        };
        let escaped_name = html_escape::encode_safe(display_name);

        let mut hasher = Sha256::new();
        hasher.update(key.as_bytes());
        let raw_hash = hasher.finalize();
        let encoded = base64::engine::general_purpose::STANDARD_NO_PAD.encode(raw_hash);
        let trip = &encoded[..10.min(encoded.len())];

        format!(r#"{escaped_name} <span class="post-tripcode">!{trip}</span>"#)
    } else {
        html_escape::encode_safe(trimmed).to_string()
    }
}

pub fn generate_unique_identifiers() -> (String, String) {
    let now = chrono::Local::now();
    let timestamp = format!(
        "{:02}/{:02}/{:02}({}){:02}:{:02}:{:02}",
        now.format("%m"),
        now.format("%d"),
        now.format("%y"),
        now.format("%a"),
        now.format("%H"),
        now.format("%M"),
        now.format("%S"),
    );

    let id = uuid::Uuid::new_v4();
    let mut hasher = sha2::Sha256::default();
    hasher.update(format!("{timestamp}{id}").as_bytes());
    let hash = format!("{:x}", hasher.finalize())[..9].to_string();

    (timestamp, hash)
}
