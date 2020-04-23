package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/alexflint/go-arg"
	"github.com/facebookgo/symwalk"
	"github.com/sirkon/errors"
	"github.com/sirkon/message"
)

func main() {
	var args struct {
		From      string   `arg:"-f,required" help:"a text to be replaced"`
		To        string   `arg:"-t" help:"a replacement text"`
		Recursive bool     `arg:"-r" help:"process directories recursively"`
		Mask      string   `arg:"-m" help:"only replace in files whose names matches this mask (wildcard)"`
		Locations []string `arg:"positional,required" help:"targets to process"`
	}

	p := arg.MustParse(&args)
	if len(args.Locations) == 0 {
		p.Fail("missing locations to process")
	}
	if args.From == "" {
		p.Fail("text to be replaced must not be empty")
	}

	var locs []string
	for _, loc := range args.Locations {
		newlocs, err := expandLocation(loc, args.Recursive)
		if err != nil {
			message.Fatal(errors.Wrap(err, "expand "+loc))
		}
		locs = append(locs, newlocs...)
	}

	// удаляем дубликаты и отфильтровываем файлы не проходящие фильтрацию по маске
	sort.Strings(locs)
	var j int
	for i := j; i < len(locs); i++ {
		if j > 0 && locs[j] == locs[j-1] {
			continue
		}
		if args.Mask != "" {
			_, file := filepath.Split(locs[i])
			match, err := filepath.Match(args.Mask, file)
			if err != nil {
				p.Fail(errors.Wrap(err, "invalid mask parameter "+args.Mask).Error())
			}
			if !match {
				continue
			}
		}

		locs[j] = locs[i]
		j++
	}
	locs = locs[:j]
	if len(locs) == 0 {
		message.Warning("no files matching a mask")
		return
	}

	from := []byte(args.From)
	to := []byte(args.To)
	// запускаем замены в файлах
	for _, loc := range locs {
		data, err := ioutil.ReadFile(loc)
		if err != nil {
			message.Fatal(errors.Wrap(err, "make replacement in "+loc))
		}

		data = bytes.ReplaceAll(data, from, to)
		if err := ioutil.WriteFile(loc, data, 0644); err != nil {
			message.Fatal(errors.Wrap(err, "save changes back into "+loc))
		}
	}
}

func expandLocation(location string, recursive bool) ([]string, error) {
	stat, err := os.Stat(location)
	if err != nil {
		return nil, errors.Wrapf(err, "read %s metainfo", location)
	}

	if !stat.IsDir() {
		return []string{location}, nil
	}

	var res []string
	if !recursive {
		files, err := ioutil.ReadDir(location)
		if err != nil {
			return nil, errors.Wrap(err, "list files in "+location)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			res = append(res, filepath.Join(location, file.Name()))
		}

		return res, nil
	}

	err = symwalk.Walk(location, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		res = append(res, filepath.Join(location, path))
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "walk %s recursively", location)
	}

	return res, nil
}
