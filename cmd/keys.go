package cmd

import (
	"PhoenixOracle/util"
	"PhoenixOracle/web/presenters"
	"fmt"
)

type CSAKeyPresenter struct {
	JAID
	presenters.CSAKeyResource
}

func (p *CSAKeyPresenter) RenderTable(rt RendererTable) error {
	headers := []string{"Public key"}
	rows := [][]string{p.ToRow()}

	if _, err := rt.Write([]byte("🔑 CSA Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return nil
}

func (p *CSAKeyPresenter) ToRow() []string {
	row := []string{
		p.PubKey,
	}

	return row
}

type CSAKeyPresenters []CSAKeyPresenter

func (ps CSAKeyPresenters) RenderTable(rt RendererTable) error {
	headers := []string{"Public key"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	if _, err := rt.Write([]byte("🔑 CSA Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)
	return utils.JustError(rt.Write([]byte("\n")))
}


type EthKeyPresenter struct {
	presenters.ETHKeyResource
}

func (p *EthKeyPresenter) ToRow() []string {
	return []string{
		p.Address,
		p.EthBalance.String(),
		p.PhbBalance.String(),
		fmt.Sprintf("%v", p.IsFunding),
		p.CreatedAt.String(),
		p.UpdatedAt.String(),
	}
}

func (p *EthKeyPresenter) RenderTable(rt RendererTable) error {
	headers := []string{"Address", "ETH", "PHB", "Is funding", "Created", "Updated"}
	rows := [][]string{p.ToRow()}

	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}

type EthKeyPresenters []EthKeyPresenter

func (ps EthKeyPresenters) RenderTable(rt RendererTable) error {
	headers := []string{"Address", "ETH", "PHB", "Is funding", "Created", "Updated"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	renderList(headers, rows, rt.Writer)

	return nil
}


type OCRKeyBundlePresenter struct {
	JAID
	presenters.OCRKeysBundleResource
}

func (p *OCRKeyBundlePresenter) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "On-chain signing addr", "Off-chain pubkey", "Config pubkey"}
	rows := [][]string{p.ToRow()}

	if _, err := rt.Write([]byte("🔑 OCR Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}

func (p *OCRKeyBundlePresenter) ToRow() []string {
	return []string{
		p.ID,
		p.OnChainSigningAddress.String(),
		p.OffChainPublicKey.String(),
		p.ConfigPublicKey.String(),
	}
}

type OCRKeyBundlePresenters []OCRKeyBundlePresenter

func (ps OCRKeyBundlePresenters) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "On-chain signing addr", "Off-chain pubkey", "Config pubkey"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	if _, err := rt.Write([]byte("🔑 OCR Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}


type P2PKeyPresenter struct {
	JAID
	presenters.P2PKeyResource
}

func (p *P2PKeyPresenter) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "Peer ID", "Public key"}
	rows := [][]string{p.ToRow()}

	if _, err := rt.Write([]byte("🔑 P2P Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}

func (p *P2PKeyPresenter) ToRow() []string {
	row := []string{
		p.ID,
		p.PeerID,
		p.PubKey,
	}

	return row
}

type P2PKeyPresenters []P2PKeyPresenter

func (ps P2PKeyPresenters) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "Peer ID", "Public key"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	if _, err := rt.Write([]byte("🔑 P2P Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}


type VRFKeyPresenter struct {
	JAID
	presenters.VRFKeyResource
}

func (p *VRFKeyPresenter) RenderTable(rt RendererTable) error {
	headers := []string{"Compressed", "Uncompressed", "Hash"}
	rows := [][]string{p.ToRow()}
	renderList(headers, rows, rt.Writer)
	_, err := rt.Write([]byte("\n"))
	return err
}

func (p *VRFKeyPresenter) ToRow() []string {
	return []string{
		p.Compressed,
		p.Uncompressed,
		p.Hash,
	}
}

type VRFKeyPresenters []VRFKeyPresenter

func (ps VRFKeyPresenters) RenderTable(rt RendererTable) error {
	headers := []string{"Compressed", "Uncompressed", "Hash"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	renderList(headers, rows, rt.Writer)
	_, err := rt.Write([]byte("\n"))
	return err
}