import 'package:movie_tool/cache/setting_cache.dart';
import 'package:tostore/tostore.dart';

SettingCache get settingCache => AppCache.instance._settingCache;

class AppCache {
  static AppCache? _instance;

  static AppCache get instance => _instance ??= AppCache._();

  AppCache._();

  late final ToStore _toStore;

  late final SettingCache _settingCache;

  Future<void> init() async {
    _toStore = await ToStore.open();

    _settingCache = SettingCache(toStore: _toStore);
  }
}
