package teamcity

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	api "github.com/yext/go-teamcity/teamcity"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectArchive,
		Importer: &schema.ResourceImporter{
			State: resourceProjectImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"parent_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"env_params": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"config_params": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"sys_params": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"config_params_specs": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"env_params_specs": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"sys_params_specs": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"feature": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"properties": {
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceProjectCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	var name, parentID string

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}

	if v, ok := d.GetOk("parent_id"); ok {
		if v != "" {
			parentID = v.(string)
		}
	}

	newProj, err := api.NewProject(name, "", parentID)
	if err != nil {
		return err
	}

	created, err := client.Projects.Create(newProj)
	if err != nil {
		return err
	}

	d.MarkNewResource()
	d.SetId(created.ID)

	return resourceProjectUpdate(d, client)
}

func resourceProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	dt, err := client.Projects.GetByID(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("name") {
		v := d.Get("name")
		err = client.Projects.Rename(d.Id(), v.(string))
		if err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("description"); ok {
		dt.Description = v.(string)
	}

	if v, ok := d.GetOk("parent_id"); ok {
		if v != "" {
			dt.SetParentProject(v.(string))
		}
	}

	if d.HasChange("feature") {
		srv := client.ProjectFeatureService(d.Id())
		err := srv.DeleteAll()
		if err != nil {
			return err
		}
		add, err := expandProjectFeatures(d.Get("feature").([]interface{}))
		if err != nil {
			return err
		}
		if len(add) > 0 {
			for i, s := range add {
				_, err := srv.Create(s)
				log.Printf("[INFO] Adding project feature '%v' with order = %v", s.Type(), i+1)
				if err != nil {
					return err
				}
			}
		}
		d.SetPartial("feature")
	}

	dt.Parameters, err = expandParameterCollection(d)
	if err != nil {
		return err
	}

	_, err = client.Projects.Update(dt)
	if err != nil {
		return nil
	}
	return resourceProjectRead(d, meta)
}

func resourceProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	dt, err := getProject(client, d.Id())
	if err != nil {
		return err
	}
	if err := d.Set("name", dt.Name); err != nil {
		return err
	}
	if err := d.Set("description", dt.Description); err != nil {
		return err
	}
	if err := d.Set("parent_id", dt.ParentProject.ID); err != nil {
		return err
	}

	err = flattenParameterCollection(d, dt.Parameters)
	if err != nil {
		return err
	}

	srv := client.ProjectFeatureService(d.Id())
	projectFeatures, err := srv.GetProjectFeatures()
	if err != nil {
		return err
	}
	projectFeaturesToSave := flattenProjectFeatures(projectFeatures)
	if err := d.Set("feature", projectFeaturesToSave); err != nil {
		return err
	}

	log.Printf("[DEBUG] Project: %v", dt)
	return nil
}

func resourceProjectArchive(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	err := client.Projects.Archive(d.Id())
	if err != nil {
		return err
	}

	name := d.Get("name")
	return client.Projects.Rename(d.Id(), fmt.Sprintf("%s (Archived)", name.(string)))
}

func resourceProjectImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceProjectRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func getProject(c *api.Client, id string) (*api.Project, error) {
	dt, err := c.Projects.GetByID(id)
	if err != nil {
		return nil, err
	}

	return dt, nil
}

func flattenProjectFeatures(pfs []api.ProjectFeature) []map[string]interface{} {
	var pfsToSave []map[string]interface{}
	for _, pf := range pfs {
		pfToSave := make(map[string]interface{})
		gpf := pf.(*api.GenericProjectFeature)
		pfToSave["type"] = gpf.Type()

		props := gpf.Properties()
		pfToSave["properties"] = make(map[string]string)
		if props != nil && props.Count > 0 {
			propertyMap := pfToSave["properties"].(map[string]string)
			for _, property := range props.Items {
				propertyMap[property.Name] = property.Value
			}
		}
		pfsToSave = append(pfsToSave, pfToSave)
	}
	return pfsToSave
}

func expandProjectFeatures(list interface{}) ([]api.ProjectFeature, error) {
	var out []api.ProjectFeature
	rawProjectFeatures := list.([]interface{})
	for _, rawPF := range rawProjectFeatures {
		expandedPF, err := expandProjectFeature(rawPF)
		if err != nil {
			return nil, err
		}
		out = append(out, expandedPF)
	}

	return out, nil
}

func expandProjectFeature(raw interface{}) (api.ProjectFeature, error) {
	feature := raw.(map[string]interface{})
	var featureType string
	var properties map[string]interface{}
	if v, _ := feature["type"]; len(v.(string)) > 0 {
		featureType = v.(string)
	} else {
		return nil, errors.New("feature type cannot be empty")
	}
	if v, ok := feature["properties"]; ok {
		properties = v.(map[string]interface{})
	}
	pf, err := api.NewGenericProjectFeature(featureType, properties)
	if err != nil {
		return nil, err
	}
	return pf, nil
}
