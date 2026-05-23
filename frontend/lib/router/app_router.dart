import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:movie_tool/page/home/home_page.dart';
import 'package:movie_tool/router/app_path.dart';

part 'app_router.g.dart';

@TypedGoRoute<HomeRoute>(path: AppPath.home)
class HomeRoute extends GoRouteData with $HomeRoute {
  @override
  Widget build(BuildContext context, GoRouterState state) => const HomePage();
}

// @TypedShellRoute<IndexRoute>(
//   routes: [
//
//   ]
// )
// class IndexRoute extends ShellRouteData {
//
//
// }

class AppRouter {
  static GoRouter route = GoRouter(
    initialLocation: AppPath.home,
    restorationScopeId: 'root',
    routes: [
      GoRoute(
        path: AppPath.home,
        builder: (context, state) => const HomePage(),
      ),
    ],
  );
}
