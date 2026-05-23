import 'package:tostore/tostore.dart';

abstract class BaseCache {
  final ToStore toStore;

  BaseCache({required this.toStore});

  Future<DbResult> setValue(
    String key,
    dynamic value, {
    Duration? ttl,
    DateTime? expiresAt,
    bool isGlobal = false,
  }) => toStore.setValue(
    key,
    value,
    ttl: ttl,
    expiresAt: expiresAt,
    isGlobal: isGlobal,
  );

  Future<dynamic> getValue(String key, {bool isGlobal = false}) =>
      toStore.getValue(key, isGlobal: isGlobal);

  Stream<T?> watchValue<T>(
    String key, {
    bool isGlobal = false,
    T? defaultValue,
    bool distinct = true,
  }) {
    return toStore.watchValue<T>(
      key,
      isGlobal: isGlobal,
      defaultValue: defaultValue,
      distinct: distinct,
    );
  }

  Future<DbResult> removeValue(String key, {bool isGlobal = false}) async {
    return await toStore.removeValue(key, isGlobal: isGlobal);
  }
}
