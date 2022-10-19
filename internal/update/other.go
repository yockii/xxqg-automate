//go:build !windows
// +build !windows

package update

func update(url string, sum []byte) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	wc := writeSumCounter{
		hash: sha256.New(),
	}
	rsp, err := io.ReadAll(io.TeeReader(resp.Body, &wc))
	if err != nil {
		return err
	}
	if !bytes.Equal(wc.hash.Sum(nil), sum) {
		return errors.New("文件已损坏")
	}
	gr, err := gzip.NewReader(bytes.NewReader(rsp))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err != nil {
			return err
		}
		if header.Name == "xxqg-automate" {
			err, _ := fromStream(tr)
			if err != nil {
				return err
			}
			return nil
		}
	}
}
