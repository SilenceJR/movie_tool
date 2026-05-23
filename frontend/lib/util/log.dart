import 'package:talker_flutter/talker_flutter.dart';

void log(
  dynamic message, {
  LogLevel logLevel = LogLevel.debug,
  Object? exception,
  StackTrace? stackTrace,
  AnsiPen? pen,
}) => Log.instance.log(
  message,
  logLevel: logLevel,
  exception: exception,
  stackTrace: stackTrace,
  pen: pen,
);

void logE(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
    Log.instance.error(msg, exception, stackTrace);

void logW(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
    Log.instance.warning(msg, exception, stackTrace);

void logD(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
    Log.instance.debug(msg, exception, stackTrace);

void logI(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
    Log.instance.info(msg, exception, stackTrace);

void logC(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
    Log.instance.critical(msg, exception, stackTrace);

void logV(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
    Log.instance.verbose(msg, exception, stackTrace);

void logH(Object exception, [StackTrace? stackTrace, dynamic msg]) =>
    Log.instance.handle(exception, stackTrace, msg);

class Log {
  static Log? _instance;

  static Log get instance => _instance ??= Log._();

  late final Talker talker;

  Log._() {
    talker = Talker();
  }

  void log(
    dynamic message, {
    LogLevel logLevel = LogLevel.debug,
    Object? exception,
    StackTrace? stackTrace,
    AnsiPen? pen,
  }) => talker.log(
    message,
    logLevel: logLevel,
    exception: exception,
    stackTrace: stackTrace,
    pen: pen,
  );

  void error(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
      talker.error(msg, exception, stackTrace);

  void warning(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
      talker.warning(msg, exception, stackTrace);

  void debug(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
      talker.debug(msg, exception, stackTrace);

  void info(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
      talker.info(msg, exception, stackTrace);

  void critical(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
      talker.critical(msg, exception, stackTrace);

  void verbose(dynamic msg, [Object? exception, StackTrace? stackTrace]) =>
      talker.verbose(msg, exception, stackTrace);

  void handle(Object exception, [StackTrace? stackTrace, dynamic msg]) =>
      talker.handle(exception, stackTrace, msg);
}
