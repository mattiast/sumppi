# Sumppi

A terminal app for fetching Supla series, generating podcast RSS feeds, and uploading them to S3.

## Feed configuration

Configuration is read from `series.toml`, or the path in `SUMPPI_CONFIG`.

```toml
[settings]
timezone = "America/New_York"

[[series]]
guid = "your-series-guid"
s3_path = "s3://your-bucket/feed.rss"
max_age_days = 60
```
