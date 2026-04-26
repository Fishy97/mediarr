import { describe, expect, test } from 'vitest';
import { groupAffectedPaths } from './pathGroups';

describe('groupAffectedPaths', () => {
  test('groups series paths by season directory', () => {
    const groups = groupAffectedPaths([
      '/media/series/The Sopranos/Season 1/The Sopranos - S01E01.mkv',
      '/media/series/The Sopranos/Season 1/The Sopranos - S01E02.mkv',
      '/media/series/The Sopranos/Season 2/The Sopranos - S02E01.mkv',
    ], 'series');

    expect(groups).toEqual([
      {
        label: 'Season 1',
        count: 2,
        paths: [
          '/media/series/The Sopranos/Season 1/The Sopranos - S01E01.mkv',
          '/media/series/The Sopranos/Season 1/The Sopranos - S01E02.mkv',
        ],
      },
      {
        label: 'Season 2',
        count: 1,
        paths: ['/media/series/The Sopranos/Season 2/The Sopranos - S02E01.mkv'],
      },
    ]);
  });

  test('uses a single file group for movie recommendations', () => {
    const groups = groupAffectedPaths(['/media/movies/Arrival (2016).mkv'], 'movie');

    expect(groups).toEqual([
      {
        label: 'File',
        count: 1,
        paths: ['/media/movies/Arrival (2016).mkv'],
      },
    ]);
  });

  test('falls back to the immediate parent directory', () => {
    const groups = groupAffectedPaths([
      '/media/anime/Cowboy Bebop/Specials/Cowboy Bebop - SP01.mkv',
      '/media/anime/Cowboy Bebop/Specials/Cowboy Bebop - SP02.mkv',
      '/media/anime/Cowboy Bebop/Cowboy Bebop - S01E01.mkv',
    ], 'anime');

    expect(groups).toEqual([
      {
        label: 'Cowboy Bebop',
        count: 1,
        paths: ['/media/anime/Cowboy Bebop/Cowboy Bebop - S01E01.mkv'],
      },
      {
        label: 'Specials',
        count: 2,
        paths: [
          '/media/anime/Cowboy Bebop/Specials/Cowboy Bebop - SP01.mkv',
          '/media/anime/Cowboy Bebop/Specials/Cowboy Bebop - SP02.mkv',
        ],
      },
    ]);
  });
});
