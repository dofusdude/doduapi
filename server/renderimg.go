package server

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

type ImageContainerResult struct {
	ContainerId string
	Resolution  string
	AnkamaId    string
}

func RenderVectorImagesWorker(swfFiles []fs.DirEntry, outPath string, ctx context.Context, resolution string, cli *client.Client, imgOutSubdirName string, done chan bool, result chan map[string]ImageContainerResult) {
	containerIds := make(map[string]ImageContainerResult)

	for _, swfFile := range swfFiles {
		select {
		case <-done:
			return
		default:
			if swfFile.IsDir() || !strings.HasSuffix(swfFile.Name(), ".swf") {
				continue
			}

			absSwfPath := fmt.Sprintf("%s/%s", outPath, swfFile.Name())
			rawFileName := strings.TrimSuffix(swfFile.Name(), ".swf")

			if _, err := os.Stat(fmt.Sprintf("data/img/%s/%s-%s.png", imgOutSubdirName, rawFileName, resolution)); err == nil {
				continue
			}

			cmd := []string{"--screenshot", "last", "--screenshot-file", fmt.Sprintf("%s-%s.png", rawFileName, resolution), "-1", "-r1", "--width", resolution, "--height", resolution, swfFile.Name()}
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image:      "stelzo/swf-renderer",
				Cmd:        cmd,
				Entrypoint: []string{"/usr/local/bin/dump-gnash"},
				Volumes: map[string]struct{}{
					"/home/developer": {},
				},
			}, &container.HostConfig{
				Binds:      []string{fmt.Sprintf("%s:/home/developer", outPath)},
				AutoRemove: true,
			}, nil, nil, "")
			if err != nil {
				panic(err)
			}

			if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
				panic(err)
			}

			containerIds[absSwfPath] = ImageContainerResult{
				ContainerId: resp.ID,
				Resolution:  resolution,
				AnkamaId:    rawFileName,
			}
		}
	}

	result <- containerIds
}

func RenderVectorImages(outPath string, done chan bool, imgOutSubdirName string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	outPath = fmt.Sprintf("%s/%s", outPath, imgOutSubdirName)

	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, "stelzo/swf-renderer", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		panic(err)
	}

	swfFiles, err := os.ReadDir(outPath)
	if err != nil {
		panic(err)
	}

	imageResolutions := []string{"800", "400", "200"}
	stopChan1 := make(chan bool)
	stopChan2 := make(chan bool)
	stopChan3 := make(chan bool)

	resChan1 := make(chan map[string]ImageContainerResult)
	resChan2 := make(chan map[string]ImageContainerResult)
	resChan3 := make(chan map[string]ImageContainerResult)

	go RenderVectorImagesWorker(swfFiles[0:5], outPath, ctx, imageResolutions[0], cli, imgOutSubdirName, stopChan1, resChan1)
	go RenderVectorImagesWorker(swfFiles[0:5], outPath, ctx, imageResolutions[1], cli, imgOutSubdirName, stopChan2, resChan2)
	go RenderVectorImagesWorker(swfFiles[0:5], outPath, ctx, imageResolutions[2], cli, imgOutSubdirName, stopChan3, resChan3)

	worker1Done := false
	worker2Done := false
	worker3Done := false

	var containerRes []map[string]ImageContainerResult

	for !worker1Done || !worker2Done || !worker3Done {
		select {
		case <-done:
			if !worker1Done {
				stopChan1 <- true
			}
			if !worker2Done {
				stopChan2 <- true
			}
			if !worker3Done {
				stopChan3 <- true
			}
			return
		case res1 := <-resChan1:
			containerRes = append(containerRes, res1)
			worker1Done = true
		case res2 := <-resChan2:
			containerRes = append(containerRes, res2)
			worker2Done = true
		case res3 := <-resChan3:
			containerRes = append(containerRes, res3)
			worker3Done = true
		}
	}

	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for _, containerRes := range containerRes {
		for absSwfPath, result := range containerRes {
			statusCh, errCh := cli.ContainerWait(ctx, result.ContainerId, container.WaitConditionRemoved)
			select {
			case err := <-errCh:
				if err != nil && !strings.Contains(err.Error(), "Error response from daemon: No such container:") {
					panic(err)
				}
			case <-statusCh:
			}

			err := os.Remove(absSwfPath)
			if err != nil {
				panic(err)
			}

			srcImagePath := fmt.Sprintf("%s/data/vector/%s/%s-%s.png", path, imgOutSubdirName, result.AnkamaId, result.Resolution)
			finalImagePath := fmt.Sprintf("%s/data/img/%s/%s-%s.png", path, imgOutSubdirName, result.AnkamaId, result.Resolution)
			err = os.Rename(srcImagePath, finalImagePath)
			if err != nil {
				log.Println(err)
			}
		}
	}

	select {
	case <-done:
		return
	}
	done <- true
}
