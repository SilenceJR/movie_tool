// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tmdb_provider.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Movie _$MovieFromJson(Map<String, dynamic> json) => _Movie(
  id: (json['id'] as num).toInt(),
  adult: json['adult'] as bool,
  originalTitle: json['original_title'] as String,
  originalLanguage: json['original_language'] as String,
  title: json['title'] as String,
  genreIds: (json['genre_ids'] as List<dynamic>)
      .map((e) => (e as num).toInt())
      .toList(),
  posterPath: json['poster_path'] as String,
  backdropPath: json['backdrop_path'] as String,
  overview: json['overview'] as String,
  releaseDate: json['release_date'] as String,
  voteAverage: (json['vote_average'] as num).toDouble(),
  voteCount: (json['vote_count'] as num).toInt(),
);

Map<String, dynamic> _$MovieToJson(_Movie instance) => <String, dynamic>{
  'id': instance.id,
  'adult': instance.adult,
  'original_title': instance.originalTitle,
  'original_language': instance.originalLanguage,
  'title': instance.title,
  'genre_ids': instance.genreIds,
  'poster_path': instance.posterPath,
  'backdrop_path': instance.backdropPath,
  'overview': instance.overview,
  'release_date': instance.releaseDate,
  'vote_average': instance.voteAverage,
  'vote_count': instance.voteCount,
};

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(PopularNotifier)
final popularProvider = PopularNotifierFamily._();

final class PopularNotifierProvider
    extends $AsyncNotifierProvider<PopularNotifier, List<Movie>> {
  PopularNotifierProvider._({
    required PopularNotifierFamily super.from,
    required Ref super.argument,
  }) : super(
         retry: null,
         name: r'popularProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$popularNotifierHash();

  @override
  String toString() {
    return r'popularProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  PopularNotifier create() => PopularNotifier();

  @override
  bool operator ==(Object other) {
    return other is PopularNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$popularNotifierHash() => r'1abce7594a6405df5d743c03022b60be207c36c7';

final class PopularNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          PopularNotifier,
          AsyncValue<List<Movie>>,
          List<Movie>,
          FutureOr<List<Movie>>,
          Ref
        > {
  PopularNotifierFamily._()
    : super(
        retry: null,
        name: r'popularProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PopularNotifierProvider call(Ref ref) =>
      PopularNotifierProvider._(argument: ref, from: this);

  @override
  String toString() => r'popularProvider';
}

abstract class _$PopularNotifier extends $AsyncNotifier<List<Movie>> {
  late final _$args = ref.$arg as Ref;
  Ref get ref => _$args;

  FutureOr<List<Movie>> build(Ref ref);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<Movie>>, List<Movie>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<Movie>>, List<Movie>>,
              AsyncValue<List<Movie>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
