use std::io::Cursor;
use image::ImageFormat;
use crate::{error::AppError, models::post::FileDetails, storage::StorageBackend};

pub const MAX_IMAGE_DIMENSION: u32 = 10_000;
pub const MAX_FILE_SIZE: usize = 32 * 1024 * 1024; // 32MB

pub struct ProcessedImageResult {
    pub details: FileDetails,
    pub main_bytes: Vec<u8>,
    pub thumb_bytes: Option<Vec<u8>>,
    pub main_filename: String,
    pub thumb_filename: Option<String>,
}

pub async fn process_upload(
    raw_bytes: Vec<u8>,
    original_filename: &str,
    unique_id: &str,
    storage: &dyn StorageBackend,
) -> Result<ProcessedImageResult, AppError> {
    if raw_bytes.len() > MAX_FILE_SIZE {
        return Err(AppError::PayloadTooLarge("File exceeds 32MB limit".into()));
    }

    let orig_name = original_filename.to_string();
    let uid = unique_id.to_string();

    let processed = tokio::task::spawn_blocking(move || -> Result<ProcessedImageResult, AppError> {
        let detected_format = image::guess_format(&raw_bytes)
            .map_err(|_| AppError::Image("Unable to determine image format or file corrupted".into()))?;

        let (format, mime) = match detected_format {
            ImageFormat::Jpeg => (ImageFormat::Jpeg, "image/jpeg"),
            ImageFormat::Png => (ImageFormat::Png, "image/png"),
            ImageFormat::Gif => (ImageFormat::Gif, "image/gif"),
            _ => return Err(AppError::Image("Unsupported image format. Allowed formats: JPEG, PNG, GIF".into())),
        };

        // 1. Fast header dimension validation without decompressing full image
        let reader = image::ImageReader::with_format(Cursor::new(&raw_bytes), format);
        let dimensions = reader
            .into_dimensions()
            .map_err(|e| AppError::Image(format!("Corrupted image header: {e}")))?;

        if dimensions.0 > MAX_IMAGE_DIMENSION || dimensions.1 > MAX_IMAGE_DIMENSION {
            return Err(AppError::Image(format!(
                "Image dimensions ({}x{}) exceed maximum {}px",
                dimensions.0, dimensions.1, MAX_IMAGE_DIMENSION
            )));
        }

        let (orig_w, orig_h) = (dimensions.0, dimensions.1);

        // 2. Fast integer sub-sampling for 250x250 thumbnail
        let dyn_img = image::load_from_memory_with_format(&raw_bytes, format)
            .map_err(|e| AppError::Image(format!("Image decode error: {e}")))?;

        let thumb_img = dyn_img.thumbnail(250, 250);
        let mut thumb_bytes = Vec::with_capacity(24 * 1024);
        let mut thumb_encoder = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut thumb_bytes, 85);
        thumb_encoder.encode_image(&thumb_img).map_err(|e| AppError::Image(e.to_string()))?;

        let clean_basename = std::path::Path::new(&orig_name)
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("upload");

        let safe_name = format!("{uid}_{clean_basename}");
        let thumb_name = format!("thumb_{safe_name}");
        let size_kb = format!("{:.1}", (raw_bytes.len() as f64) / 1024.0);
        let dim_str = format!("{orig_w}x{orig_h}");

        Ok(ProcessedImageResult {
            details: FileDetails {
                name: safe_name.clone(),
                path: String::new(),
                mime: mime.to_string(),
                size: size_kb,
                dimensions: dim_str,
            },
            main_bytes: raw_bytes,
            thumb_bytes: Some(thumb_bytes),
            main_filename: safe_name,
            thumb_filename: Some(thumb_name),
        })
    })
    .await
    .map_err(|e| AppError::Internal(anyhow::anyhow!("Image worker task panicked: {e}")))??;

    // 3. Parallel asynchronous storage writes
    if let (Some(thumb_data), Some(thumb_name)) = (&processed.thumb_bytes, &processed.thumb_filename) {
        tokio::try_join!(
            storage.save(&processed.main_filename, &processed.main_bytes),
            storage.save(thumb_name, thumb_data)
        )?;
    } else {
        storage.save(&processed.main_filename, &processed.main_bytes).await?;
    }

    let mut result = processed;
    result.details.path = storage.public_url(&result.details.name);
    Ok(result)
}
