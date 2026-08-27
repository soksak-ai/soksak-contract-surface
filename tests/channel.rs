use soksak_contract_surface::channel_name;

#[test]
fn the_channel_name_derives_from_the_identifier() {
    assert_eq!(channel_name("com.soksak.wails.perfanalysis").unwrap(), "com.soksak.wails.perfanalysis.surface");
}

#[test]
fn an_overlong_or_empty_identifier_is_refused_with_the_limit_named() {
    let error = channel_name(&"a".repeat(128)).unwrap_err();
    assert!(error.contains("128"), "{error}");
    assert!(channel_name("").is_err());
}
