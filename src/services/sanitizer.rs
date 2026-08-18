use std::sync::LazyLock;
use regex::Regex;

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
    use sha2::Digest;
    hasher.update(format!("{timestamp}{id}").as_bytes());
    let hash = format!("{:x}", hasher.finalize())[..9].to_string();

    (timestamp, hash)
}
