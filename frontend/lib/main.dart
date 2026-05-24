import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:movie_tool/cache/cache.dart';
import 'package:movie_tool/router/app_router.dart';
import 'package:movie_tool/util/log.dart';
import 'package:talker_riverpod_logger/talker_riverpod_logger.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  await AppCache.instance.init();

  runApp(
    ProviderScope(
      observers: [TalkerRiverpodObserver(talker: Log.instance.talker)],
      child: MyApp(),
    ),
  );
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Flutter Demo',
      theme: ThemeData(colorScheme: .fromSeed(seedColor: Colors.deepPurple)),
      routerConfig: AppRouter.route,
      restorationScopeId: 'App',
    );
  }
}
