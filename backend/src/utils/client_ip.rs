use axum::http::HeaderMap;
use std::net::IpAddr;

fn parse_ip(value: &str) -> Option<IpAddr> {
    value.trim().trim_matches('"').parse().ok()
}

fn is_private_ip(ip: &IpAddr) -> bool {
    match ip {
        IpAddr::V4(v4) => {
            let octets = v4.octets();
            octets[0] == 10
                || (octets[0] == 172 && (16..=31).contains(&octets[1]))
                || (octets[0] == 192 && octets[1] == 168)
        }
        IpAddr::V6(v6) => v6.is_loopback() || v6.is_unicast_link_local(),
    }
}

fn parse_x_forwarded_for(header: &str) -> Option<IpAddr> {
    for part in header.split(',') {
        let token = part.trim();
        let ip_part = token.split(':').next().unwrap_or(token);
        if let Some(ip) = parse_ip(ip_part) {
            if !is_private_ip(&ip) {
                return Some(ip);
            }
        }
    }

    header
        .split(',')
        .next()
        .and_then(|first| parse_ip(first.split(':').next().unwrap_or(first.trim())))
}

fn parse_forwarded(header: &str) -> Option<IpAddr> {
    for part in header.split(';') {
        let token = part.trim();
        if let Some(rest) = token.strip_prefix("for=") {
            let content = rest.trim().trim_matches('"');
            if content.starts_with('[') {
                let end = content.find(']')?;
                return parse_ip(&content[1..end]);
            }
            let ip_part = content.split(':').next().unwrap_or(content);
            return parse_ip(ip_part);
        }
    }
    None
}

/// 获取客户端 IP，遵循 spec/api_spec_base.md 定义的优先级
pub fn get_client_ip_from_headers(headers: &HeaderMap) -> String {
    if let Some(value) = headers
        .get("CF-Connecting-IP")
        .and_then(|v| v.to_str().ok())
        .and_then(parse_ip)
    {
        return value.to_string();
    }

    if let Some(value) = headers
        .get("x-real-ip")
        .and_then(|v| v.to_str().ok())
        .and_then(parse_ip)
    {
        return value.to_string();
    }

    if let Some(value) = headers
        .get("x-forwarded-for")
        .and_then(|v| v.to_str().ok())
        .and_then(parse_x_forwarded_for)
    {
        return value.to_string();
    }

    if let Some(value) = headers
        .get("forwarded")
        .and_then(|v| v.to_str().ok())
        .and_then(parse_forwarded)
    {
        return value.to_string();
    }

    "unknown".to_string()
}
